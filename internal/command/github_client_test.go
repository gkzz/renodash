package command

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
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

// TestRunGHFailure checks that gh failures preserve the exit error while omitting diagnostics.
func TestRunGHFailure(t *testing.T) {
	installFakeGH(t, "failure", "sensitive diagnostic")
	_, err := runGH(context.Background(), "api", "graphql")
	if err == nil || err.Error() != "gh command failed: exit status 1" {
		t.Fatalf("error=%v", err)
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("error does not wrap exit error: %v", err)
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
			_, err := (ghClient{}).FetchPR(context.Background(), "owner/repo", 4)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("err=%v want=%q", err, tc.want)
			}
		})
	}
	t.Run("missing executable", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		_, err := (ghClient{}).FetchPR(context.Background(), "owner/repo", 4)
		if err == nil || err.Error() != "gh is not installed or not in PATH" {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("canceled", func(t *testing.T) {
		installFakeGH(t, "output", "")
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := (ghClient{}).FetchPR(ctx, "owner/repo", 4)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestGHClientCancelsRunningProcess(t *testing.T) {
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	client := ghClient{}
	for _, tc := range []struct {
		name  string
		fetch func(context.Context, string, int) ([]byte, error)
	}{
		{"saved token", func(ctx context.Context, _ string, _ int) ([]byte, error) {
			_, err := issueToken(ctx)
			return nil, err
		}},
		{"PR", client.FetchPR},
	} {
		t.Run(tc.name, func(t *testing.T) {
			listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			if err := listener.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
				t.Fatal(err)
			}
			installFakeGH(t, "wait", listener.Addr().String())
			ctx, cancel := context.WithCancel(context.Background())
			var output []byte
			var fetchErr error
			done := make(chan struct{})
			go func() {
				defer close(done)
				output, fetchErr = tc.fetch(ctx, "owner/repo", 4)
			}()
			defer func() {
				cancel()
				select {
				case <-done:
				case <-time.After(5 * time.Second):
					t.Error("fetch did not finish during cleanup")
				}
			}()
			conn, err := listener.Accept()
			if err != nil {
				t.Fatalf("wait for helper startup: %v", err)
			}
			defer conn.Close()
			if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
				t.Fatal(err)
			}
			ready := make([]byte, len("ready\n"))
			if _, err := io.ReadFull(conn, ready); err != nil || string(ready) != "ready\n" {
				t.Fatalf("startup signal=%q err=%v", ready, err)
			}
			select {
			case <-done:
				t.Fatal("fetch returned before cancellation")
			default:
			}
			cancel()
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("running helper was not canceled within the cleanup bound")
			}
			if output != nil || !errors.Is(fetchErr, context.Canceled) {
				t.Fatalf("output=%q err=%v; want nil output and context.Canceled", output, fetchErr)
			}
			// Returning a context error alone must not leave the helper alive.
			if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
				t.Fatal(err)
			}
			var extra [1]byte
			if n, err := conn.Read(extra[:]); n != 0 || err != io.EOF {
				t.Fatalf("helper connection remains open: n=%d err=%v", n, err)
			}
		})
	}
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

func TestRunMissingIssueCredentials(t *testing.T) {
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("PATH", t.TempDir())
	var out, diagnostic bytes.Buffer
	code := Run([]string{"get", "owner/repo#4"}, strings.NewReader(""), &out, &diagnostic)
	want := "error: fetch owner/repo#4: GitHub authentication failed; set GH_TOKEN or GITHUB_TOKEN, or run gh auth login\n"
	if code != 1 || out.Len() != 0 || diagnostic.String() != want {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), diagnostic.String())
	}
}
