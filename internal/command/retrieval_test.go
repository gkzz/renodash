package command

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	dashpkg "github.com/gkzz/renodash/internal/dashboard"
)

func TestPRRetries(t *testing.T) {
	for _, tc := range []struct {
		name      string
		failure   error
		wantCalls int
	}{
		{"timeout", context.DeadlineExceeded, 3},
		{"server", &requestError{code: "http_5xx", retryable: true}, 3},
		{"rate limit", &requestError{code: "rate_limit", retryable: true, wait: 3 * time.Second}, 3},
		{"authentication", &requestError{code: "authentication"}, 1},
		{"permission", &requestError{code: "permission_denied"}, 1},
		{"unknown", errors.New("HTTP 503 diagnostic is not structured evidence"), 1},
		{"cancellation", context.Canceled, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls, waits := 0, 0
			client := fakeGitHubClient{prFn: func(ctx context.Context, _ string, _ int) ([]byte, error) {
				if _, ok := ctx.Deadline(); !ok {
					t.Fatal("missing attempt deadline")
				}
				calls++
				return nil, tc.failure
			}}
			policy := retryPolicy{delay: func(attempt int) time.Duration { return time.Duration(attempt) * time.Second }, wait: func(_ context.Context, d time.Duration) error {
				waits++
				if tc.name == "rate limit" && d != 3*time.Second {
					t.Fatalf("wait=%s", d)
				}
				return nil
			}}
			_, attempts, err := retryPR(context.Background(), client, "owner/repo", 1, policy)
			if !errors.Is(err, tc.failure) || calls != tc.wantCalls || attempts != calls || waits != calls-1 {
				t.Fatalf("calls=%d attempts=%d waits=%d err=%v", calls, attempts, waits, err)
			}
		})
	}
}

func TestPRRetryRecoveryAndDeduplication(t *testing.T) {
	calls := map[int]int{}
	client := fakeGitHubClient{prFn: func(_ context.Context, _ string, n int) ([]byte, error) {
		calls[n]++
		if n == 33 && calls[n] == 1 {
			return nil, context.DeadlineExceeded
		}
		return []byte(fmt.Sprintf(`{"number":%d,"url":"https://github.com/owner/repo/pull/%d","state":"OPEN","headRefOid":"a","updatedAt":"time"}`, n, n)), nil
	}}
	policy := retryPolicy{delay: func(int) time.Duration { return 0 }, wait: func(context.Context, time.Duration) error { return nil }}
	prs, retrieval, err := collectPullRequests(context.Background(), client, "owner/repo", []dashpkg.Item{{PRLink: "../pull/32"}, {PRLink: "../pull/33"}, {PRLink: "pull/033"}}, true, policy)
	if err != nil || len(prs) != 2 || prs[0].Number != 32 || prs[1].Number != 33 || calls[32] != 1 || calls[33] != 2 || retrieval.Status != "complete" || len(retrieval.Errors) != 0 {
		t.Fatalf("prs=%+v retrieval=%+v calls=%v err=%v", prs, retrieval, calls, err)
	}
}
func TestRetryBudgetAndCancellation(t *testing.T) {
	for _, where := range []string{"budget", "request", "backoff", "retry-after", "expired"} {
		t.Run(where, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if where == "expired" {
				cancel()
			}
			calls, waits := 0, 0
			client := fakeGitHubClient{prFn: func(context.Context, string, int) ([]byte, error) {
				calls++
				if where == "request" {
					cancel()
					return []byte("late success"), nil
				}
				if where == "retry-after" {
					return nil, &requestError{code: "rate_limit", retryable: true, wait: 500 * time.Millisecond}
				}
				return nil, context.DeadlineExceeded
			}}
			policy := retryPolicy{delay: func(int) time.Duration {
				if where == "budget" {
					return time.Hour
				}
				return 0
			}, wait: func(ctx context.Context, d time.Duration) error {
				waits++
				if where == "retry-after" && d != 500*time.Millisecond {
					t.Fatalf("wait=%s", d)
				}
				cancel()
				return waitForRetry(ctx, d)
			}}
			data, _, err := retryPR(ctx, client, "owner/repo", 1, policy)
			if data != nil || err == nil || calls > 1 {
				t.Fatalf("data=%s calls=%d err=%v", data, calls, err)
			}
			if where == "budget" {
				if waits != 0 || !errors.Is(err, context.DeadlineExceeded) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !errors.Is(waitForRetry(ctx, time.Hour), context.Canceled) {
		t.Fatal("wait ignored cancellation")
	}
}
func TestPartialCollectionFailures(t *testing.T) {
	for _, tc := range []struct{ name, body, code string }{
		{"fetch", "", "fetch_failed"}, {"decode", "{", "decode_error"}, {"type", `{"number":"bad"}`, "decode_error"}, {"metadata", "{}", "invalid_metadata"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := fakeGitHubClient{prFn: func(context.Context, string, int) ([]byte, error) {
				calls++
				if tc.name == "fetch" {
					return nil, errors.New("secret")
				}
				return []byte(tc.body), nil
			}}
			items := []dashpkg.Item{{PRLink: "../pull/33"}, {PRLink: "pull/033"}, {PRLink: "https://other.invalid/pull/1"}, {PRLink: "https://other.invalid/pull/1"}}
			prs, r, err := collectPullRequests(context.Background(), client, "owner/repo", items, true, defaultRetryPolicy())
			if err != nil || len(prs) != 0 || prs == nil || calls != 1 || r.Status != "partial" || len(r.Errors) != 2 || r.Errors[0].Code != tc.code || r.Errors[0].Attempts != 1 || r.Errors[1].Code != "invalid_link" || r.Errors[1].Attempts != 0 || r.Errors[1].Number != 0 {
				t.Fatalf("prs=%v r=%+v calls=%d err=%v", prs, r, calls, err)
			}
			data, _ := json.Marshal(r)
			if strings.Contains(string(data), "secret") {
				t.Fatal(string(data))
			}
		})
	}
}
func TestBackoffJitterBounds(t *testing.T) {
	p := defaultRetryPolicy()
	for i := 0; i < 100; i++ {
		for attempt := 1; attempt <= 2; attempt++ {
			d := p.delay(attempt)
			ceiling := time.Second * time.Duration(1<<(attempt-1))
			if d < ceiling/2 || d > ceiling {
				t.Fatalf("attempt=%d delay=%s", attempt, d)
			}
		}
	}
}

func TestPartialRateLimitStopsCollection(t *testing.T) {
	for _, tc := range []struct {
		name     string
		wait     time.Duration
		retry    bool
		attempts int
	}{
		{"wait exceeds budget", 10 * time.Minute, true, 1},
		{"retry limit", time.Second, true, 3},
		{"unknown wait", 0, false, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			calls := map[int]int{}
			client := fakeGitHubClient{prFn: func(_ context.Context, _ string, n int) ([]byte, error) {
				calls[n]++
				if n == 2 {
					return nil, &requestError{code: "rate_limit", retryable: tc.retry, wait: tc.wait}
				}
				return []byte(fmt.Sprintf(`{"number":%d,"url":"https://github.com/owner/repo/pull/%d","state":"OPEN","headRefOid":"a","updatedAt":"time"}`, n, n)), nil
			}}
			policy := retryPolicy{delay: func(int) time.Duration { return 0 }, wait: func(context.Context, time.Duration) error { return nil }}
			prs, r, err := collectPullRequests(ctx, client, "owner/repo", []dashpkg.Item{{PRLink: "pull/1"}, {PRLink: "pull/2"}, {PRLink: "pull/3"}, {PRLink: "pull/03"}, {PRLink: "pull/4"}}, true, policy)
			if err != nil || len(prs) != 1 || calls[2] != tc.attempts || calls[3] != 0 || calls[4] != 0 || r.Status != "partial" || len(r.Errors) != 3 {
				t.Fatalf("prs=%v retrieval=%+v calls=%v err=%v", prs, r, calls, err)
			}
			for i, e := range r.Errors {
				wantAttempts := 0
				if i == 0 {
					wantAttempts = tc.attempts
				}
				if e.Number != i+2 || e.Code != "rate_limit" || e.Attempts != wantAttempts || e.Retryable != tc.retry {
					t.Fatalf("error=%+v", e)
				}
			}
		})
	}
}
