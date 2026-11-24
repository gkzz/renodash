package command

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	dashpkg "github.com/gkzz/renodash/internal/dashboard"
)

type retrievalResult struct {
	Status string           `json:"status"`
	Errors []retrievalError `json:"errors"`
}
type retrievalError struct {
	Resource   string `json:"resource"`
	Repository string `json:"repository"`
	Number     int    `json:"number,omitempty"`
	Link       string `json:"link,omitempty"`
	Code       string `json:"code"`
	Attempts   int    `json:"attempts"`
	Retryable  bool   `json:"retryable"`
}

// Transport errors contain only a stable category and a server-specified wait.
// Raw headers, response bodies, and diagnostics never enter error messages.
type requestError struct {
	code      string
	retryable bool
	wait      time.Duration
}

func (e *requestError) Error() string { return "GitHub request failed: " + e.code }

func requestFailure(err error) (string, bool, time.Duration) {
	var request *requestError
	if errors.As(err, &request) {
		return request.code, request.retryable, request.wait
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout", true, 0
	}
	var timeout interface{ Timeout() bool }
	if errors.As(err, &timeout) && timeout.Timeout() {
		return "timeout", true, 0
	}
	return "fetch_failed", false, 0
}

type retryPolicy struct {
	wait  func(context.Context, time.Duration) error
	delay func(int) time.Duration
}

func defaultRetryPolicy() retryPolicy {
	return retryPolicy{wait: waitForRetry, delay: func(attempt int) time.Duration {
		ceiling := time.Second * time.Duration(1<<(attempt-1))
		return ceiling/2 + time.Duration(rand.Int64N(int64(ceiling/2)+1))
	}}
}
func waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return ctx.Err()
	}
}
func retryPR(ctx context.Context, client githubClient, repo string, number int, policy retryPolicy) ([]byte, int, error) {
	for attempt := 1; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, attempt - 1, err
		}
		data, err := fetchPR(ctx, client, repo, number)
		if ctx.Err() != nil {
			return nil, attempt, ctx.Err()
		}
		if err == nil {
			return data, attempt, nil
		}
		_, retryable, minimumWait := requestFailure(err)
		if !retryable || attempt == 3 {
			return nil, attempt, err
		}
		delay := max(policy.delay(attempt), minimumWait)
		if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) <= delay {
			return nil, attempt, err
		}
		if err := policy.wait(ctx, delay); err != nil {
			return nil, attempt, err
		}
	}
}

func collectPullRequests(ctx context.Context, client githubClient, repo string, items []dashpkg.Item, partial bool, policy retryPolicy) ([]pullRequestMetadata, retrievalResult, error) {
	prs := []pullRequestMetadata{}
	retrieval := retrievalResult{Status: "complete", Errors: []retrievalError{}}
	seen := map[int]bool{}
	invalid := map[string]bool{}
	fail := func(e retrievalError) { retrieval.Status = "partial"; retrieval.Errors = append(retrieval.Errors, e) }
	for _, item := range items {
		if err := ctx.Err(); err != nil {
			return nil, retrieval, err
		}
		if item.PRLink == "" {
			continue
		}
		number, err := linkedPRNumber(item.PRLink, repo)
		if err != nil {
			if !partial {
				return nil, retrieval, err
			}
			if !invalid[item.PRLink] {
				fail(retrievalError{Resource: "pull_request", Repository: repo, Link: item.PRLink, Code: "invalid_link"})
				invalid[item.PRLink] = true
			}
			continue
		}
		if seen[number] {
			continue
		}
		seen[number] = true
		data, attempts, err := retryPR(ctx, client, repo, number, policy)
		if ctx.Err() != nil {
			return nil, retrieval, ctx.Err()
		}
		if errors.Is(err, context.Canceled) {
			return nil, retrieval, err
		}
		if err != nil {
			if !partial {
				return nil, retrieval, fmt.Errorf("fetch PR %s#%d: %w", repo, number, err)
			}
			code, retryable, _ := requestFailure(err)
			fail(retrievalError{Resource: "pull_request", Repository: repo, Number: number, Code: code, Attempts: attempts, Retryable: retryable})
			continue
		}
		pr, err := decodePullRequest(data, repo, number, time.Now().UTC().Format(time.RFC3339))
		if err != nil {
			if !partial {
				return nil, retrieval, err
			}
			code := "invalid_metadata"
			if errors.Is(err, errPRDecode) {
				code = "decode_error"
			}
			fail(retrievalError{Resource: "pull_request", Repository: repo, Number: number, Code: code, Attempts: attempts})
			continue
		}
		prs = append(prs, pr)
	}
	if err := ctx.Err(); err != nil {
		return nil, retrieval, err
	}
	return prs, retrieval, nil
}
