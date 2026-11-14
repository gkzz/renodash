package command

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Each test owns its callbacks; unexpected operations fail immediately.
type fakeGitHubClient struct {
	issueFn func(context.Context, string, int) ([]byte, error)
	prFn    func(context.Context, string, int) ([]byte, error)
}

func (f fakeGitHubClient) FetchIssue(ctx context.Context, repo string, number int) ([]byte, error) {
	return f.issueFn(ctx, repo, number)
}
func (f fakeGitHubClient) FetchPR(ctx context.Context, repo string, number int) ([]byte, error) {
	return f.prFn(ctx, repo, number)
}

// TestExecuteGHFailure checks that gh failures preserve the original exit error without exposing diagnostics.
func TestExecuteGHFailure(t *testing.T) {
	installFakeGH(t, "failure", "sensitive diagnostic")
	for _, args := range [][]string{{"api", "repos/gkzz/example/issues/4"}, {"pr", "view", "33", "--repo", "gkzz/example"}} {
		t.Run(strings.Join(args[:2], " "), func(t *testing.T) {
			_, err := executeGH(context.Background(), args...)
			if err == nil || err.Error() != "gh command failed: exit status 1" {
				t.Fatalf("error=%v", err)
			}
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) {
				t.Fatalf("error does not wrap exit error: %v", err)
			}
		})
	}
}

func TestGHClientArguments(t *testing.T) {
	installFakeGH(t, "arguments", "")
	client := ghClient{}
	for _, tc := range []struct {
		name  string
		fetch func(context.Context, string, int) ([]byte, error)
		want  string
	}{
		{"issue", client.FetchIssue, "api\nrepos/Owner/Repo/issues/4\n"},
		{"PR", client.FetchPR, "pr\nview\n4\n--repo\nOwner/Repo\n--json\nnumber,url,state,title,isDraft,headRefOid,updatedAt,statusCheckRollup\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.fetch(context.Background(), "Owner/Repo", 4)
			if err != nil || string(got) != tc.want {
				t.Fatalf("output=%q err=%v want=%q", got, err, tc.want)
			}
		})
	}
}

func TestGHClientErrors(t *testing.T) {
	for _, tc := range []struct{ name, diagnostic, want string }{
		{"authentication", "Authentication failed token=secret", "gh authentication failed; run gh auth login"},
		{"not logged in", "not logged in token=secret", "gh authentication failed; run gh auth login"},
		{"401", "HTTP 401 token=secret", "gh authentication failed; run gh auth login"},
		{"404", "HTTP 404 token=secret", "issue not found or inaccessible"},
		{"not found", "not found token=secret", "issue not found or inaccessible"},
		{"other", "token=secret", "gh command failed: exit status 1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			installFakeGH(t, "failure", tc.diagnostic)
			client := ghClient{}
			for _, fetch := range []func(context.Context, string, int) ([]byte, error){client.FetchIssue, client.FetchPR} {
				_, err := fetch(context.Background(), "owner/repo", 4)
				if err == nil || err.Error() != tc.want {
					t.Fatalf("err=%v want=%q", err, tc.want)
				}
			}
		})
	}
	t.Run("missing executable", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		_, err := (ghClient{}).FetchIssue(context.Background(), "owner/repo", 4)
		if err == nil || err.Error() != "gh is not installed or not in PATH" {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("canceled", func(t *testing.T) {
		installFakeGH(t, "output", "")
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := (ghClient{}).FetchIssue(ctx, "owner/repo", 4)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err=%v", err)
		}
	})
}

// A client returning data after cancellation must not turn the request into success.
func TestFetchRejectsLateSuccess(t *testing.T) {
	for _, operation := range []string{"issue", "PR"} {
		for _, expired := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/expired=%t", operation, expired), func(t *testing.T) {
				var parent context.Context
				var cancel context.CancelFunc
				want := context.Canceled
				if expired {
					parent, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
					want = context.DeadlineExceeded
				} else {
					parent, cancel = context.WithCancel(context.Background())
				}
				defer cancel()
				var request context.Context
				callback := func(ctx context.Context, _ string, _ int) ([]byte, error) {
					request = ctx
					cancel()
					return []byte(`{}`), nil
				}
				client := fakeGitHubClient{issueFn: callback, prFn: callback}
				fetch := fetchIssue
				if operation == "PR" {
					fetch = fetchPR
				}
				data, err := fetch(parent, client, "owner/repo", 4)
				if !errors.Is(err, want) || data != nil || request.Err() == nil {
					t.Fatalf("data=%s err=%v request=%v", data, err, request.Err())
				}
			})
		}
	}
}

func TestRunUsesGHClient(t *testing.T) {
	installFakeGH(t, "output", `{"body":"## Open","number":4,"html_url":"https://github.com/owner/repo/issues/4","updated_at":"2026-11-03T12:42:39Z"}`)
	var out, diagnostic bytes.Buffer
	if code := Run([]string{"get", "owner/repo#4"}, strings.NewReader(""), &out, &diagnostic); code != 0 || !strings.Contains(out.String(), `"issue_number": 4`) {
		t.Fatalf("stdout=%s stderr=%s", out.String(), diagnostic.String())
	}
}

func TestRunGHFailureLeavesStdoutEmpty(t *testing.T) {
	installFakeGH(t, "failure", "HTTP 401 token=secret")
	var out, diagnostic bytes.Buffer
	code := Run([]string{"get", "owner/repo#4"}, strings.NewReader(""), &out, &diagnostic)
	want := "error: fetch owner/repo#4: gh authentication failed; run gh auth login\n"
	if code != 1 || out.Len() != 0 || diagnostic.String() != want {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), diagnostic.String())
	}
}
