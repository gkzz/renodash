package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/gkzz/renodash"
)

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
	if code := run([]string{"get", "gkzz/example#4"}, strings.NewReader(""), &out, &diagnostic); code != 0 {
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

func TestGetEscapesNewlineInWarning(t *testing.T) {
	oldGH, oldParse := runGH, parseGetDashboard
	t.Cleanup(func() { runGH, parseGetDashboard = oldGH, oldParse })
	runGH = func(context.Context, string) ([]byte, error) {
		return []byte(`{"body":"## Open","number":4,"html_url":"https://github.com/gkzz/example/issues/4","updated_at":"2026-11-03T12:42:39Z"}`), nil
	}
	parseGetDashboard = func(io.Reader) (renodash.Dashboard, error) {
		return renodash.Dashboard{UnknownControls: []string{"first\nsecond"}}, nil
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

func TestGetErrors(t *testing.T) {
	oldGH := runGH
	t.Cleanup(func() { runGH = oldGH })
	for _, reference := range []string{"", "owner/repo", "owner/repo#0", "owner/repo#-1", "owner/repo#1/2", "owner//repo#1", "../repo#1", "owner/repo#1#2", "owner/repo?x#1"} {
		var out, diagnostic bytes.Buffer
		if code := run([]string{"get", reference}, strings.NewReader(""), &out, &diagnostic); code == 0 || out.Len() != 0 {
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
			if code := run([]string{"get", "gkzz/example#4"}, strings.NewReader(""), &out, &diagnostic); code == 0 || out.Len() != 0 || !strings.Contains(diagnostic.String(), tc.want) {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), diagnostic.String())
			}
		})
	}
}

func TestGetParseError(t *testing.T) {
	oldGH, oldParse := runGH, parseGetDashboard
	t.Cleanup(func() { runGH, parseGetDashboard = oldGH, oldParse })
	runGH = func(context.Context, string) ([]byte, error) {
		return []byte(`{"body":"## Open","number":4,"html_url":"https://github.com/gkzz/example/issues/4","updated_at":"2026-11-03T12:42:39Z"}`), nil
	}
	parseGetDashboard = func(_ io.Reader) (renodash.Dashboard, error) { return renodash.Dashboard{}, errors.New("parse failed") }
	var out, diagnostic bytes.Buffer
	if code := run([]string{"get", "gkzz/example#4"}, strings.NewReader(""), &out, &diagnostic); code == 0 || out.Len() != 0 || !strings.Contains(diagnostic.String(), "parse failed") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), diagnostic.String())
	}
}

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
	if code := run([]string{"get", "GKZZ/RENODASH#5"}, strings.NewReader(""), &out, &diagnostic); code != 0 {
		t.Fatalf("code=%d stderr=%q", code, diagnostic.String())
	}
	if !strings.Contains(out.String(), `"repository": "gkzz/renodash"`) {
		t.Fatal(out.String())
	}
}

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
	if code := run([]string{"get", "gkzz/example#4"}, strings.NewReader(""), &out, &diagnostic); code == 0 || out.Len() != 0 || !strings.Contains(diagnostic.String(), "context deadline exceeded") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), diagnostic.String())
	}
}
