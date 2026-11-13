package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	dashpkg "github.com/gkzz/renodash/internal/dashboard"
)

// TestExecuteGHFailure checks that gh failures preserve the original exit error without exposing diagnostics.
func TestExecuteGHFailure(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte("#!/bin/sh\necho 'sensitive diagnostic' >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
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

// TestGet checks the issue endpoint, JSON output with source metadata, and warnings for unsupported controls.
func TestGet(t *testing.T) {
	oldGH, oldNow := runGH, getNow
	t.Cleanup(func() { runGH, getNow = oldGH, oldNow })
	getNow = func() time.Time { return time.Date(2026, 11, 4, 1, 2, 3, 0, time.FixedZone("JST", 9*3600)) }
	runGH = func(_ context.Context, endpoint string) ([]byte, error) {
		if endpoint != "repos/gkzz/example/issues/4" {
			t.Fatalf("endpoint=%q", endpoint)
		}
		return []byte(`{"body":"## Open\n- [ ] <!-- rebase-branch=renovate/a --> [Update a](../pull/33)\n- [ ] <!-- future --> Unknown\n","number":4,"html_url":"https://github.com/gkzz/example/issues/4","updated_at":"2026-11-03T12:42:39Z"}`), nil
	}
	var out, diagnostic bytes.Buffer
	if code := Run([]string{"get", "gkzz/example#4"}, strings.NewReader(""), &out, &diagnostic); code != 0 {
		t.Fatalf("code=%d stderr=%q", code, diagnostic.String())
	}
	for _, want := range []string{`"repository": "gkzz/example"`, `"issue_number": 4`, `"fetched_at": "2026-11-03T16:02:03Z"`, `"issue_updated_at": "2026-11-03T12:42:39Z"`, `"pr_link": "../pull/33"`, `"bulk_actions": []`, `"unknown_controls": [`} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q in %s", want, out.String())
		}
	}
	if !strings.Contains(diagnostic.String(), `warning: unsupported Renovate control: "future"`) {
		t.Fatal(diagnostic.String())
	}
}

// TestGetEscapesWarningsBeforeJSON checks that control characters in warnings are escaped and all warnings precede JSON output.
// Unsupported controls in JSON retain their original strings.
func TestGetEscapesWarningsBeforeJSON(t *testing.T) {
	oldGH := runGH
	t.Cleanup(func() { runGH = oldGH })
	body := "## Open\n- [ ] <!-- future\x1b[31m --> Unknown\n- [ ] <!-- another\rcontrol --> Unknown\n"
	issue := githubIssue{Body: &body, Number: 4, HTMLURL: "https://github.com/gkzz/example/issues/4", UpdatedAt: "2026-11-03T12:42:39Z"}
	data, err := json.Marshal(issue)
	if err != nil {
		t.Fatal(err)
	}
	runGH = func(context.Context, string) ([]byte, error) { return data, nil }
	var out, diagnostic bytes.Buffer
	var writes []string
	if err := executeGet([]string{"gkzz/example#4"}, recordingWriter{&out, &writes, "stdout"}, recordingWriter{&diagnostic, &writes, "stderr"}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(writes, ","); got != "stderr,stderr,stdout" {
		t.Fatalf("write order=%s", got)
	}
	want := "warning: unsupported Renovate control: \"future\\x1b[31m\"\n" +
		"warning: unsupported Renovate control: \"another\\rcontrol\"\n"
	if diagnostic.String() != want {
		t.Fatalf("stderr=%q want=%q", diagnostic.String(), want)
	}
	if strings.ContainsAny(diagnostic.String(), "\x1b\r") || strings.Count(diagnostic.String(), "\n") != 2 {
		t.Fatalf("unsafe warning output: %q", diagnostic.String())
	}
	var result getResult
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.UnknownControls) != 2 || result.UnknownControls[0] != "future\x1b[31m" || result.UnknownControls[1] != "another\rcontrol" {
		t.Fatalf("unknown_controls=%q", result.UnknownControls)
	}
}

// TestGetWarningWriteFailureLeavesStdoutEmpty checks that a warning write failure returns an error and leaves stdout empty.
func TestGetWarningWriteFailureLeavesStdoutEmpty(t *testing.T) {
	oldGH := runGH
	t.Cleanup(func() { runGH = oldGH })
	runGH = func(context.Context, string) ([]byte, error) {
		return []byte(`{"body":"## Open\n- [ ] <!-- future --> Unknown\n","number":4,"html_url":"https://github.com/gkzz/example/issues/4","updated_at":"2026-11-03T12:42:39Z"}`), nil
	}
	var out bytes.Buffer
	err := executeGet([]string{"gkzz/example#4"}, &out, failingWriter{})
	if err == nil || !strings.Contains(err.Error(), "write warning") || out.Len() != 0 {
		t.Fatalf("err=%v stdout=%q", err, out.String())
	}
}

// TestGetEscapesNewlineInWarning checks that newlines in unsupported controls are escaped to keep each warning on one line.
func TestGetEscapesNewlineInWarning(t *testing.T) {
	oldGH, oldParse := runGH, parseGetDashboard
	t.Cleanup(func() { runGH, parseGetDashboard = oldGH, oldParse })
	runGH = func(context.Context, string) ([]byte, error) {
		return []byte(`{"body":"## Open","number":4,"html_url":"https://github.com/gkzz/example/issues/4","updated_at":"2026-11-03T12:42:39Z"}`), nil
	}
	parseGetDashboard = func(io.Reader) (dashpkg.Dashboard, error) {
		return dashpkg.Dashboard{UnknownControls: []string{"first\nsecond"}}, nil
	}
	var out, diagnostic bytes.Buffer
	if err := executeGet([]string{"gkzz/example#4"}, &out, &diagnostic); err != nil {
		t.Fatal(err)
	}
	if got, want := diagnostic.String(), "warning: unsupported Renovate control: \"first\\nsecond\"\n"; got != want {
		t.Fatalf("stderr=%q want=%q", got, want)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

type recordingWriter struct {
	output *bytes.Buffer
	writes *[]string
	name   string
}

func (w recordingWriter) Write(p []byte) (int, error) {
	*w.writes = append(*w.writes, w.name)
	return w.output.Write(p)
}

// TestGetErrors checks that invalid references, fetch failures, invalid JSON, empty bodies,
// and mismatched metadata cause failures and leave stdout empty.
func TestGetErrors(t *testing.T) {
	oldGH := runGH
	t.Cleanup(func() { runGH = oldGH })
	for _, reference := range []string{"", "owner/repo", "owner/repo#0", "owner/repo#-1", "owner/repo#1/2", "owner//repo#1", "../repo#1", "owner/repo#1#2", "owner/repo?x#1"} {
		var out, diagnostic bytes.Buffer
		if code := Run([]string{"get", reference}, strings.NewReader(""), &out, &diagnostic); code == 0 || out.Len() != 0 {
			t.Errorf("reference=%q code=%d stdout=%q", reference, code, out.String())
		}
	}
	for _, tc := range []struct {
		name, response string
		err            error
		want           string
	}{
		{"gh failure", "", errors.New("gh failed"), "gh failed"},
		{"bad JSON", "{", nil, "decode GitHub issue JSON"},
		{"empty body", `{"body":"  ","number":4,"html_url":"https://github.com/gkzz/example/issues/4","updated_at":"2026-11-03T12:42:39Z"}`, nil, "body is empty"},
		{"bad metadata", `{"body":"abc","number":5,"html_url":"https://github.com/gkzz/example/issues/4","updated_at":"2026-11-03T12:42:39Z"}`, nil, "mismatched metadata"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runGH = func(context.Context, string) ([]byte, error) { return []byte(tc.response), tc.err }
			var out, diagnostic bytes.Buffer
			if code := Run([]string{"get", "gkzz/example#4"}, strings.NewReader(""), &out, &diagnostic); code == 0 || out.Len() != 0 || !strings.Contains(diagnostic.String(), tc.want) {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), diagnostic.String())
			}
		})
	}
}

// TestGetParseError checks that Dashboard parse errors are reported and stdout stays empty.
func TestGetParseError(t *testing.T) {
	oldGH, oldParse := runGH, parseGetDashboard
	t.Cleanup(func() { runGH, parseGetDashboard = oldGH, oldParse })
	runGH = func(context.Context, string) ([]byte, error) {
		return []byte(`{"body":"## Open","number":4,"html_url":"https://github.com/gkzz/example/issues/4","updated_at":"2026-11-03T12:42:39Z"}`), nil
	}
	parseGetDashboard = func(_ io.Reader) (dashpkg.Dashboard, error) { return dashpkg.Dashboard{}, errors.New("parse failed") }
	var out, diagnostic bytes.Buffer
	if code := Run([]string{"get", "gkzz/example#4"}, strings.NewReader(""), &out, &diagnostic); code == 0 || out.Len() != 0 || !strings.Contains(diagnostic.String(), "parse failed") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), diagnostic.String())
	}
}

// TestGetIssueURLCase checks that case differences between the input and fetched URL are accepted
// and the output uses the repository casing from the fetched URL.
func TestGetIssueURLCase(t *testing.T) {
	oldGH := runGH
	t.Cleanup(func() { runGH = oldGH })
	runGH = func(_ context.Context, endpoint string) ([]byte, error) {
		if endpoint != "repos/GKZZ/RENODASH/issues/5" {
			t.Fatalf("endpoint=%q", endpoint)
		}
		return []byte(`{"body":"## Open","number":5,"html_url":"https://github.com/gkzz/renodash/issues/5","updated_at":"2026-11-03T12:42:39Z"}`), nil
	}
	var out, diagnostic bytes.Buffer
	if code := Run([]string{"get", "GKZZ/RENODASH#5"}, strings.NewReader(""), &out, &diagnostic); code != 0 {
		t.Fatalf("code=%d stderr=%q", code, diagnostic.String())
	}
	if !strings.Contains(out.String(), `"repository": "gkzz/renodash"`) {
		t.Fatal(out.String())
	}
}

// TestMatchesIssueURL checks that an HTTPS URL points to the issue in the target repository.
// Host and repository names are case-insensitive; other destinations, queries, and extra URL components are rejected.
func TestMatchesIssueURL(t *testing.T) {
	for _, tc := range []struct {
		url  string
		want bool
	}{
		{"https://github.com/gkzz/renodash/issues/5", true},
		{"https://GITHUB.COM/GKZZ/RENODASH/issues/5", true},
		{"http://github.com/gkzz/renodash/issues/5", false},
		{"https://github.com.evil/gkzz/renodash/issues/5", false},
		{"https://github.com:443/gkzz/renodash/issues/5", false},
		{"https://github.com/gkzz/other/issues/5", false},
		{"https://github.com/gkzz/renodash/Issues/5", false},
		{"https://github.com/gkzz/renodash/issues/05", false},
		{"https://github.com/gkzz/renodash/issues/6", false},
		{"https://github.com/gkzz/renodash/issues/5?x=1", false},
		{"https://github.com/gkzz/renodash/issues/5#x", false},
		{"https://github.com/gkzz/%72enodash/issues/5", false},
	} {
		if got := matchesIssueURL(tc.url, "gkzz/renodash", 5); got != tc.want {
			t.Errorf("url=%q got=%t want=%t", tc.url, got, tc.want)
		}
	}
}

// TestGetTimeoutLeavesStdoutEmpty checks that fetching has a deadline and timeouts report an error while leaving stdout empty.
func TestGetTimeoutLeavesStdoutEmpty(t *testing.T) {
	oldGH := runGH
	t.Cleanup(func() { runGH = oldGH })
	runGH = func(ctx context.Context, _ string) ([]byte, error) {
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("missing deadline")
		}
		return []byte(`{"body":"partial"}`), context.DeadlineExceeded
	}
	var out, diagnostic bytes.Buffer
	if code := Run([]string{"get", "gkzz/example#4"}, strings.NewReader(""), &out, &diagnostic); code == 0 || out.Len() != 0 || !strings.Contains(diagnostic.String(), "context deadline exceeded") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), diagnostic.String())
	}
}

// TestParseIssueRef checks the syntax and return values for OWNER/REPO#ISSUE_NUMBER.
// It validates owner and repository characters, the 100-character repository limit, positive integer bounds,
// and leading zeros. Invalid input returns an error, an empty repository name, and issue number zero.
func TestParseIssueRef(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantRepo   string
		wantNumber int
		want       bool
	}{
		{"personal account", "octocat/Hello-World#1", "octocat/Hello-World", 1, true},
		{"organization", "octo-org/repo#2", "octo-org/repo", 2, true},
		{"owner trailing hyphen", "org-/repo#1", "org-/repo", 1, true},
		{"owner consecutive hyphens", "org--name/repo#1", "org--name/repo", 1, true},
		{"owner leading hyphen compatibility", "-org/repo#1", "-org/repo", 1, true},
		{"managed user", "mona-cat_octo/repo#1", "mona-cat_octo/repo", 1, true},
		{"owner length not restricted locally", strings.Repeat("a", 40) + "/repo#1", strings.Repeat("a", 40) + "/repo", 1, true},
		{"repository punctuation", "owner/._repo--_ .#1", "", 0, false},
		{"repository leading punctuation", "owner/._repo--_.#1", "owner/._repo--_.", 1, true},
		{"repository triple dot", "owner/...#1", "owner/...", 1, true},
		{"repository git suffix retained", "owner/repo.git#1", "owner/repo.git", 1, true},
		{"repository wiki suffix retained", "owner/repo.wiki#1", "owner/repo.wiki", 1, true},
		{"ASCII digits", "123/456#1", "123/456", 1, true},
		{"NUL", "owner/repo\x00#1", "", 0, false},
		{"maximum int", "owner/repo#" + strconv.Itoa(int(^uint(0)>>1)), "owner/repo", int(^uint(0) >> 1), true},
		{"repository hyphen", "owner/-#1", "owner/-", 1, true},
		{"repository underscore", "owner/_#1", "owner/_", 1, true},
		{"repository maximum", "owner/" + strings.Repeat("a", 100) + "#1", "owner/" + strings.Repeat("a", 100), 1, true},
		{"repository too long", "owner/" + strings.Repeat("a", 101) + "#1", "", 0, false},
		{"owner dot", "owner.name/repo#1", "", 0, false},
		{"owner triple dot", ".../repo#1", "", 0, false},
		{"empty owner", "/repo#1", "", 0, false},
		{"empty repository", "owner/#1", "", 0, false},
		{"repository dot", "owner/.#1", "", 0, false},
		{"repository double dot", "owner/..#1", "", 0, false},
		{"unicode owner", "所有者/repo#1", "", 0, false},
		{"unicode repository", "owner/リポジトリ#1", "", 0, false},
		{"non ASCII letter", "owner/café#1", "", 0, false},
		{"space", "owner/re po#1", "", 0, false},
		{"control", "owner/repo\n#1", "", 0, false},
		{"escaped path", "owner/repo%2fother#1", "", 0, false},
		{"query", "owner/repo?x#1", "", 0, false},
		{"missing separator", "owner/repo", "", 0, false},
		{"extra slash", "owner/repo/other#1", "", 0, false},
		{"empty number", "owner/repo#", "", 0, false},
		{"zero", "owner/repo#0", "", 0, false},
		{"negative", "owner/repo#-1", "", 0, false},
		{"plus sign", "owner/repo#+1", "", 0, false},
		{"unicode digit", "owner/repo#１", "", 0, false},
		{"extra hash", "owner/repo#1#2", "", 0, false},
		{"overflow", "owner/repo#" + strings.Repeat("9", 100), "", 0, false},
		{"leading zeros", "OWNER/Repo#0001", "OWNER/Repo", 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, number, err := parseIssueRef(tt.input)
			if got := err == nil; got != tt.want {
				t.Errorf("parseIssueRef(%q) success=%t, want %t (error=%v)", tt.input, got, tt.want, err)
			}
			if tt.want && err == nil {
				if repo != tt.wantRepo || number != tt.wantNumber {
					t.Errorf("got (%q, %d), want (%q, %d)", repo, number, tt.wantRepo, tt.wantNumber)
				}
			}
			if err != nil && (repo != "" || number != 0) {
				t.Errorf("error returned partial result (%q, %d)", repo, number)
			}
		})
	}
}
