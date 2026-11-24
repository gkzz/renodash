package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestGetStdinAndPartial(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		args        []string
		success     bool
	}{
		{"partial", "gkzz/example#4\r\n", []string{"--with-pr-metadata", "--allow-partial", "--stdin"}, true},
		{"empty", " \n", []string{"--stdin"}, false},
		{"multiple", "gkzz/example#4\ngkzz/example#5", []string{"--stdin"}, false},
		{"conflict", "gkzz/example#4", []string{"--stdin", "gkzz/example#4"}, false},
		{"requires metadata", "gkzz/example#4", []string{"--stdin", "--allow-partial"}, false},
		{"oversized", strings.Repeat("a", 4097), []string{"--stdin"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := fakeGitHubClient{issueFn: func(context.Context, string, int) ([]byte, error) {
				calls++
				return []byte(`{"body":"## Open\n- [ ] <!-- rebase-branch=a --> [Update a](../pull/33)\n- [ ] <!-- rebase-branch=b --> [Update b](../pull/34)","number":4,"html_url":"https://github.com/gkzz/example/issues/4","updated_at":"2026-11-03T12:42:39Z"}`), nil
			}, prFn: func(_ context.Context, _ string, n int) ([]byte, error) {
				if n == 33 {
					return nil, errors.New("secret diagnostic")
				}
				return []byte(`{"number":34,"url":"https://github.com/gkzz/example/pull/34","state":"OPEN","headRefOid":"abc","updatedAt":"2026-11-03T12:42:39Z"}`), nil
			}}
			var out, diag bytes.Buffer
			code := runWithClient(client, append([]string{"get"}, tc.args...), strings.NewReader(tc.input), &out, &diag)
			if !tc.success {
				if code == 0 || calls != 0 || out.Len() != 0 {
					t.Fatalf("code=%d calls=%d output=%s", code, calls, &out)
				}
				return
			}
			if code != 0 {
				t.Fatalf("code=%d diagnostic=%s", code, &diag)
			}
			var result struct {
				PullRequests []struct {
					Number int `json:"number"`
				} `json:"pull_requests"`
				Retrieval struct {
					Status string `json:"status"`
					Errors []struct {
						Number, Attempts int
						Code             string
						Retryable        bool
					} `json:"errors"`
				} `json:"retrieval"`
			}
			if err := json.Unmarshal(out.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.Retrieval.Status != "partial" || len(result.Retrieval.Errors) != 1 || result.Retrieval.Errors[0].Number != 33 || result.Retrieval.Errors[0].Attempts != 1 || result.Retrieval.Errors[0].Retryable || len(result.PullRequests) != 1 || result.PullRequests[0].Number != 34 || strings.Contains(out.String(), "secret") {
				t.Fatalf("result=%s", &out)
			}
		})
	}
}

func TestGetStdinPlainAndFailures(t *testing.T) {
	for _, mode := range []string{"plain", "no PRs", "read failure", "dashboard failure", "write failure"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			client := fakeGitHubClient{issueFn: func(context.Context, string, int) ([]byte, error) {
				calls++
				if mode == "dashboard failure" {
					return nil, errors.New("unavailable")
				}
				return []byte(`{"body":"## Open","number":4,"html_url":"https://github.com/gkzz/example/issues/4","updated_at":"2026-11-03T12:42:39Z"}`), nil
			}}
			args := []string{"get", "--stdin"}
			if mode != "plain" {
				args = append(args, "--with-pr-metadata", "--allow-partial")
			}
			var input io.Reader = strings.NewReader(" gkzz/example#4\n")
			if mode == "read failure" {
				input = errorReader{}
			}
			var out, diag bytes.Buffer
			var output io.Writer = &out
			if mode == "write failure" {
				output = failingWriter{}
			}
			code := runWithClient(client, args, input, output, &diag)
			if mode == "plain" || mode == "no PRs" {
				if code != 0 || calls != 1 {
					t.Fatalf("code=%d calls=%d diagnostic=%s", code, calls, &diag)
				}
				var result map[string]any
				if err := json.Unmarshal(out.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				if mode == "plain" {
					if _, ok := result["retrieval"]; ok {
						t.Fatal("plain get changed schema")
					}
				} else {
					r := result["retrieval"].(map[string]any)
					if r["status"] != "complete" || len(r["errors"].([]any)) != 0 {
						t.Fatalf("result=%s", &out)
					}
				}
			} else if code == 0 || out.Len() != 0 {
				t.Fatalf("code=%d output=%s", code, &out)
			}
			if mode == "read failure" && calls != 0 {
				t.Fatal("requested issue after stdin read error")
			}
		})
	}
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, errors.New("input failed") }
