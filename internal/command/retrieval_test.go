package command

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
	for _, where := range []string{"budget", "request", "backoff", "expired"} {
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
				return nil, context.DeadlineExceeded
			}}
			policy := retryPolicy{delay: func(int) time.Duration {
				if where == "budget" {
					return time.Hour
				}
				return 0
			}, wait: func(ctx context.Context, _ time.Duration) error { waits++; cancel(); return ctx.Err() }}
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
func TestStructuredHTTPFailures(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		status  int
		headers string
		code    string
		retry   bool
	}{
		{200, "", "", false}, {401, "", "authentication", false}, {403, "", "permission_denied", false}, {404, "", "not_found_or_inaccessible", false}, {503, "", "http_5xx", true}, {429, "Retry-After: 5\r\n", "rate_limit", true}, {403, "X-RateLimit-Remaining: 0\r\nX-RateLimit-Reset: " + fmt.Sprint(now.Add(time.Minute).Unix()) + "\r\n", "rate_limit", true}, {429, "", "rate_limit", false}, {429, "Retry-After: nonsense\r\n", "rate_limit", false},
	} {
		t.Run(fmt.Sprint(tc.status, tc.headers), func(t *testing.T) {
			body, header, err := readGHResponse([]byte(fmt.Sprintf("HTTP/2.0 %d Status\r\n%s\r\n{\"secret\":true}", tc.status, tc.headers)))
			if err != nil || string(body) != `{"secret":true}` {
				t.Fatalf("body=%s err=%v", body, err)
			}
			err = httpFailure(header)
			if tc.code == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			code, retry, _ := requestFailure(err)
			if code != tc.code || retry != tc.retry || strings.Contains(err.Error(), "secret") {
				t.Fatalf("code=%s retry=%t err=%v", code, retry, err)
			}
		})
	}
	h := http.Header{"Retry-After": []string{now.Add(10 * time.Second).UTC().Format(http.TimeFormat)}}
	if wait, ok := rateLimitWait(h, now); !ok || wait < 9*time.Second || wait > 10*time.Second {
		t.Fatalf("wait=%s ok=%t", wait, ok)
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
