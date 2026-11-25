package command

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Fetch the same check fields as gh pr view, with explicit HTTP metadata.
// See https://github.com/cli/cli/blob/v2.97.0/api/query_builder.go.
const prQuery = `query($owner:String!,$repo:String!,$number:Int!,$cursor:String){repository(owner:$owner,name:$repo){pullRequest(number:$number){number url state title isDraft headRefOid updatedAt mergeable mergeStateStatus commits(last:1){nodes{commit{oid statusCheckRollup{contexts(first:100,after:$cursor){nodes{__typename ... on StatusContext{context state targetUrl createdAt description} ... on CheckRun{name checkSuite{workflowRun{workflow{name}}} status conclusion startedAt completedAt detailsUrl}} pageInfo{hasNextPage endCursor}}}}}}}}}`

func (ghClient) FetchPR(ctx context.Context, repo string, number int) ([]byte, error) {
	owner, name, _ := strings.Cut(repo, "/")
	cursor := ""
	seen := map[string]bool{}
	var first map[string]json.RawMessage
	var checks []json.RawMessage
	for {
		args := []string{"api", "graphql", "--hostname", "github.com", "--include", "-f", "query=" + prQuery, "-f", "owner=" + owner, "-f", "repo=" + name, "-F", "number=" + strconv.Itoa(number)}
		if cursor != "" {
			args = append(args, "-f", "cursor="+cursor)
		}
		output, processErr := runGH(ctx, args...)
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		body, header, err := readGHResponse(output)
		if err != nil {
			if processErr != nil {
				return nil, processErr
			}
			return nil, err
		}
		if err := httpFailure(header, body); err != nil {
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
		if processErr != nil {
			return nil, processErr
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
	checks, err := normalizeChecks(checks)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(checks)
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

// gh --include prints status/headers without HTTP chunk framing. Its body is
// already decoded by gh, so net/http.ReadResponse would misread HTTP/2/chunks.
func readGHResponse(output []byte) ([]byte, http.Header, error) {
	reader := bufio.NewReader(bytes.NewReader(output))
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, nil, errors.New("missing GitHub HTTP status")
	}
	fields := strings.Fields(line)
	if len(fields) < 2 || !strings.HasPrefix(fields[0], "HTTP/") {
		return nil, nil, errors.New("invalid GitHub HTTP status")
	}
	status, err := strconv.Atoi(fields[1])
	if err != nil || status < 100 || status > 599 {
		return nil, nil, errors.New("invalid GitHub HTTP status")
	}
	header := http.Header{}
	header.Set("X-Renodash-Status", strconv.Itoa(status))
	for {
		line, err = reader.ReadString('\n')
		if err != nil {
			return nil, nil, errors.New("invalid GitHub HTTP headers")
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, nil, errors.New("invalid GitHub HTTP headers")
		}
		// Never permit a response header to override our parsed status.
		if strings.EqualFold(key, "X-Renodash-Status") {
			continue
		}
		header.Add(key, strings.TrimSpace(value))
	}
	body := new(bytes.Buffer)
	_, err = body.ReadFrom(reader)
	return body.Bytes(), header, err
}

func httpFailure(header http.Header, body []byte) error {
	status, _ := strconv.Atoi(header.Get("X-Renodash-Status"))
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

// Inspect only structured API errors, never subprocess diagnostics. Messages are
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
