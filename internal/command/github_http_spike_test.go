package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const nativeSpikeOrigin = "https://api.github.com"

type nativeSpikeClient struct {
	http  *http.Client
	token string
}

var _ githubClient = nativeSpikeClient{}

func newNativeSpike() nativeSpikeClient {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	return nativeSpikeClient{http: &http.Client{
		Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}}
}

func (c nativeSpikeClient) FetchIssue(ctx context.Context, repo string, number int) ([]byte, error) {
	if _, _, err := parseIssueRef(fmt.Sprintf("%s#%d", repo, number)); err != nil {
		return nil, err
	}
	body, header, err := c.request(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/issues/%d", repo, number), nil)
	if err != nil {
		return nil, err
	}
	if err := httpFailure(header, body); err != nil {
		return nil, err
	}
	return body, nil
}

func (c nativeSpikeClient) FetchPR(ctx context.Context, repo string, number int) ([]byte, error) {
	if _, _, err := parseIssueRef(fmt.Sprintf("%s#%d", repo, number)); err != nil {
		return nil, err
	}
	owner, name, _ := strings.Cut(repo, "/")
	return readPRPages(func(cursor string) ([]byte, http.Header, error, error) {
		variables := map[string]any{"owner": owner, "repo": name, "number": number}
		if cursor != "" {
			variables["cursor"] = cursor
		}
		payload, err := json.Marshal(struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}{prQuery, variables})
		if err != nil {
			return nil, nil, nil, &requestError{code: "encode_error"}
		}
		body, header, err := c.request(ctx, http.MethodPost, "/graphql", bytes.NewReader(payload))
		return body, header, nil, err
	})
}

func (c nativeSpikeClient) request(ctx context.Context, method, path string, payload io.Reader) ([]byte, http.Header, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	token := c.token
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		return nil, nil, &requestError{code: "authentication"}
	}
	req, err := http.NewRequestWithContext(ctx, method, nativeSpikeOrigin+path, payload)
	if err != nil {
		return nil, nil, &requestError{code: "request_failed"}
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
		// Keep POST bodies from being automatically replayed by the transport.
		req.GetBody = nil
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, nil, nativeSpikeIOError(ctx, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return nil, nil, &requestError{code: "redirect"}
	}
	// Use only the status from the HTTP response line.
	resp.Header.Set("X-Renodash-Status", strconv.Itoa(resp.StatusCode))
	body, err := io.ReadAll(resp.Body)
	ctxErr := ctx.Err()
	if ctxErr == context.Canceled {
		return nil, nil, ctxErr
	}
	if err != nil || ctxErr != nil {
		failure := httpFailure(resp.Header, body)
		if failure == nil && path == "/graphql" && structuredRateLimit(body) {
			wait, known := rateLimitWait(resp.Header, time.Now())
			failure = &requestError{code: "rate_limit", retryable: known, wait: wait}
		}
		// A per-attempt deadline must preserve any known rate limit.
		code, _, _ := requestFailure(failure)
		if failure != nil && (ctxErr == nil || code == "rate_limit") {
			return nil, nil, failure
		}
		return nil, nil, nativeSpikeIOError(ctx, err)
	}
	return body, resp.Header, nil
}

func nativeSpikeIOError(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	var networkError net.Error
	if errors.Is(err, context.DeadlineExceeded) || errors.As(err, &networkError) && networkError.Timeout() {
		return &requestError{code: "timeout", retryable: true}
	}
	return &requestError{code: "fetch_failed"}
}
