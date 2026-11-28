package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Fetch the same check fields as gh pr view, with explicit HTTP metadata.
// See https://github.com/cli/cli/blob/v2.97.0/api/query_builder.go.
const prQuery = `query($owner:String!,$repo:String!,$number:Int!,$cursor:String){repository(owner:$owner,name:$repo){pullRequest(number:$number){number url state title isDraft headRefOid updatedAt mergeable mergeStateStatus commits(last:1){nodes{commit{oid statusCheckRollup{contexts(first:100,after:$cursor){nodes{__typename ... on StatusContext{context state targetUrl createdAt description} ... on CheckRun{name checkSuite{workflowRun{workflow{name}}} status conclusion startedAt completedAt detailsUrl}} pageInfo{hasNextPage endCursor}}}}}}}}}`

type graphQLVariables struct {
	Owner  string `json:"owner"`
	Repo   string `json:"repo"`
	Number int    `json:"number"`
	Cursor string `json:"cursor,omitempty"`
}

type graphQLRequest struct {
	Query     string           `json:"query"`
	Variables graphQLVariables `json:"variables"`
}

func (c *githubHTTPClient) fetchPRPage(ctx context.Context, token string, variables graphQLVariables) ([]byte, http.Header, error) {
	payload, err := json.Marshal(graphQLRequest{Query: prQuery, Variables: variables})
	if err != nil {
		return nil, nil, errors.New("encode GitHub PR request")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.github.com/graphql", bytes.NewReader(payload))
	if err != nil {
		return nil, nil, errors.New("create GitHub PR request")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "renodash")
	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, nil, ctx.Err()
		}
		var timeout interface{ Timeout() bool }
		if errors.As(err, &timeout) && timeout.Timeout() {
			return nil, nil, &requestError{code: "timeout", retryable: true}
		}
		return nil, nil, errors.New("GitHub PR request failed")
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(resp.Body)
	if ctx.Err() != nil {
		return nil, nil, ctx.Err()
	}
	// Preserve the HTTP classification even if an error response is truncated.
	if err := httpFailure(resp.StatusCode, resp.Header, body); err != nil {
		return nil, nil, err
	}
	if readErr != nil {
		return nil, nil, &requestError{code: "decode_error"}
	}
	return body, resp.Header, nil
}

func (c *githubHTTPClient) FetchPR(ctx context.Context, repo string, number int) ([]byte, error) {
	token, err := c.credentials(ctx)
	if err != nil {
		return nil, err
	}
	owner, name, _ := strings.Cut(repo, "/")
	cursor := ""
	seen := map[string]bool{}
	var first map[string]json.RawMessage
	var checks []json.RawMessage
	for {
		body, header, err := c.fetchPRPage(ctx, token, graphQLVariables{
			Owner: owner, Repo: name, Number: number, Cursor: cursor,
		})
		if err != nil {
			return nil, err
		}
		var response struct {
			Errors []struct {
				Type string `json:"type"`
			} `json:"errors"`
			Data struct {
				Repository *struct {
					PullRequest map[string]json.RawMessage `json:"pullRequest"`
				} `json:"repository"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, &requestError{code: "decode_error"}
		}
		if len(response.Errors) > 0 {
			code := "graphql_error"
			if structuredRateLimit(body) {
				code = "rate_limit"
			}
			wait, known := rateLimitWait(header, time.Now())
			return nil, &requestError{code: code, retryable: code == "rate_limit" && known, wait: wait}
		}
		if response.Data.Repository == nil || response.Data.Repository.PullRequest == nil {
			return nil, &requestError{code: "not_found_or_inaccessible"}
		}
		pr := response.Data.Repository.PullRequest
		var commits struct {
			Nodes []struct {
				Commit struct {
					OID    string `json:"oid"`
					Rollup *struct {
						Contexts struct {
							Nodes    []json.RawMessage `json:"nodes"`
							PageInfo *struct {
								HasNextPage *bool  `json:"hasNextPage"`
								EndCursor   string `json:"endCursor"`
							} `json:"pageInfo"`
						} `json:"contexts"`
					} `json:"statusCheckRollup"`
				} `json:"commit"`
			} `json:"nodes"`
		}
		commitData, ok := pr["commits"]
		if !ok {
			return nil, &requestError{code: "invalid_metadata"}
		}
		if err := json.Unmarshal(commitData, &commits); err != nil {
			return nil, &requestError{code: "decode_error"}
		}
		if len(commits.Nodes) != 1 || commits.Nodes[0].Commit.OID == "" {
			return nil, &requestError{code: "invalid_metadata"}
		}
		if first == nil {
			first = pr
		} else if !bytes.Equal(first["headRefOid"], pr["headRefOid"]) {
			return nil, &requestError{code: "head_changed"}
		}
		if cursor != "" && commits.Nodes[0].Commit.Rollup == nil {
			return nil, &requestError{code: "invalid_pagination"}
		}
		more, next := false, ""
		for _, node := range commits.Nodes {
			var head string
			if err := json.Unmarshal(pr["headRefOid"], &head); err != nil || node.Commit.OID != head {
				return nil, &requestError{code: "head_changed"}
			}
			if node.Commit.Rollup != nil {
				c := node.Commit.Rollup.Contexts
				if c.PageInfo == nil || c.PageInfo.HasNextPage == nil {
					return nil, &requestError{code: "invalid_pagination"}
				}
				if c.Nodes == nil {
					return nil, &requestError{code: "invalid_metadata"}
				}
				checks = append(checks, c.Nodes...)
				more, next = *c.PageInfo.HasNextPage, c.PageInfo.EndCursor
			}
		}
		if !more {
			break
		}
		if next == "" || seen[next] {
			return nil, &requestError{code: "invalid_pagination"}
		}
		seen[next] = true
		cursor = next
	}
	if checks == nil {
		checks = []json.RawMessage{}
	}
	normalized, err := normalizeChecks(checks)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return nil, err
	}
	first["statusCheckRollup"] = encoded
	delete(first, "commits")
	return json.Marshal(first)
}

// Match the JSON exported by gh pr view, rather than exposing GraphQL nodes.
// See https://github.com/cli/cli/blob/v2.97.0/api/export_pr.go.
func normalizeChecks(nodes []json.RawMessage) ([]json.RawMessage, error) {
	checks := make([]json.RawMessage, 0, len(nodes))
	for _, node := range nodes {
		var c struct {
			Type                                 string `json:"__typename"`
			Name, Status, Conclusion, DetailsURL string
			StartedAt, CompletedAt, CreatedAt    time.Time
			Context, State, TargetURL            string
			CheckSuite                           struct {
				WorkflowRun struct{ Workflow struct{ Name string } }
			}
		}
		if err := json.Unmarshal(node, &c); err != nil {
			return nil, &requestError{code: "decode_error"}
		}
		var check map[string]any
		switch c.Type {
		case "CheckRun":
			check = map[string]any{"__typename": c.Type, "name": c.Name, "workflowName": c.CheckSuite.WorkflowRun.Workflow.Name, "status": c.Status, "conclusion": c.Conclusion, "startedAt": c.StartedAt, "completedAt": c.CompletedAt, "detailsUrl": c.DetailsURL}
		case "StatusContext":
			check = map[string]any{"__typename": c.Type, "context": c.Context, "state": c.State, "targetUrl": c.TargetURL, "startedAt": c.CreatedAt}
		default:
			return nil, &requestError{code: "invalid_metadata"}
		}
		data, err := json.Marshal(check)
		if err != nil {
			return nil, err
		}
		checks = append(checks, data)
	}
	return checks, nil
}

func httpFailure(status int, header http.Header, body []byte) error {
	if status >= 200 && status < 300 {
		return nil
	}
	if status == 401 {
		return &requestError{code: "authentication"}
	}
	if status == 429 || status == 403 && (header.Get("Retry-After") != "" || header.Get("X-RateLimit-Remaining") == "0" || structuredRateLimit(body)) {
		wait, known := rateLimitWait(header, time.Now())
		return &requestError{code: "rate_limit", retryable: known, wait: wait}
	}
	if status == 403 {
		return &requestError{code: "permission_denied"}
	}
	if status == 404 {
		return &requestError{code: "not_found_or_inaccessible"}
	}
	if status >= 500 {
		return &requestError{code: "http_5xx", retryable: true}
	}
	return &requestError{code: "http_error"}
}

// Inspect structured API errors. Messages are
// used for classification only and must not be included in returned errors.
func structuredRateLimit(body []byte) bool {
	var response struct {
		Message string `json:"message"`
		Errors  []struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	if json.Unmarshal(body, &response) != nil {
		return false
	}
	if strings.Contains(strings.ToLower(response.Message), "secondary rate limit") {
		return true
	}
	for _, e := range response.Errors {
		if e.Type == "RATE_LIMITED" || strings.Contains(strings.ToLower(e.Message), "secondary rate limit") {
			return true
		}
	}
	return false
}

func rateLimitWait(header http.Header, now time.Time) (time.Duration, bool) {
	if value := header.Get("Retry-After"); value != "" {
		if seconds, err := strconv.ParseInt(value, 10, 32); err == nil && seconds >= 0 {
			return time.Duration(seconds) * time.Second, true
		}
		if date, err := http.ParseTime(value); err == nil {
			return max(0, date.Sub(now)), true
		}
		return 0, false
	}
	if header.Get("X-RateLimit-Remaining") == "0" {
		if seconds, err := strconv.ParseInt(header.Get("X-RateLimit-Reset"), 10, 64); err == nil {
			return max(0, time.Unix(seconds, 0).Sub(now)), true
		}
	}
	return 0, false
}
