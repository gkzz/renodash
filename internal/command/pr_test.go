package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	dashpkg "github.com/gkzz/renodash/internal/dashboard"
)

func TestLinkedPRNumber(t *testing.T) {
	for _, link := range []string{"../pull/33", "pull/33", "/gkzz/example/pull/33", "https://github.com/GKZZ/example/pull/33"} {
		if n, err := linkedPRNumber(link, "gkzz/example"); err != nil || n != 33 {
			t.Errorf("%q: number=%d err=%v", link, n, err)
		}
	}
	for _, link := range []string{"https://evil.test/gkzz/example/pull/33", "https://github.com/other/repo/pull/33", "../pull/0", "../pull/-1", "../pull/33?x=1", "../pull/33#x", "//evil.test/pull/33", "../pull/33/files"} {
		if _, err := linkedPRNumber(link, "gkzz/example"); err == nil {
			t.Errorf("accepted %q", link)
		}
	}
}

func TestGetWithPRMetadata(t *testing.T) {
	oldGH, oldPR := runGH, runGHPR
	t.Cleanup(func() { runGH, runGHPR = oldGH, oldPR })
	runGH = func(context.Context, string) ([]byte, error) {
		return []byte(`{"body":"## Open\n- [ ] <!-- rebase-branch=renovate/a --> [Update a](../pull/33)\n- [ ] <!-- rebase-branch=renovate/b --> [Update b](../pull/33)","number":4,"html_url":"https://github.com/gkzz/example/issues/4","updated_at":"2026-11-03T12:42:39Z"}`), nil
	}
	calls := 0
	runGHPR = func(_ context.Context, repo string, number int) ([]byte, error) {
		calls++
		if repo != "gkzz/example" || number != 33 {
			t.Fatalf("repo=%s number=%d", repo, number)
		}
		return []byte(`{"number":33,"url":"https://github.com/gkzz/example/pull/33","state":"OPEN","title":"Update a","isDraft":true,"headRefOid":"abc123","updatedAt":"2026-11-03T12:42:39Z","statusCheckRollup":[{"__typename":"CheckRun","name":"test","status":"COMPLETED","conclusion":"FAILURE"},{"__typename":"StatusContext","context":"external","state":"PENDING"}]}`), nil
	}
	var out, diagnostic bytes.Buffer
	if code := Run([]string{"get", "--with-pr-metadata", "gkzz/example#4"}, strings.NewReader(""), &out, &diagnostic); code != 0 {
		t.Fatalf("stderr=%s", diagnostic.String())
	}
	var result getResult
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || result.PullRequests == nil || len(*result.PullRequests) != 1 {
		t.Fatalf("calls=%d result=%s", calls, out.String())
	}
	pr := (*result.PullRequests)[0]
	if pr.State != "OPEN" || !pr.IsDraft || pr.HeadRefOID != "abc123" || len(pr.Checks) != 2 {
		t.Fatalf("pr=%+v", pr)
	}
	runGHPR = func(context.Context, string, int) ([]byte, error) { return nil, errors.New("unavailable") }
	out.Reset()
	if err := executeGet([]string{"gkzz/example#4", "--with-pr-metadata"}, &out, &diagnostic); err == nil || out.Len() != 0 {
		t.Fatalf("err=%v output=%s", err, out.String())
	}
}

func TestPullRequestStatesAndInvalidMetadata(t *testing.T) {
	oldPR := runGHPR
	t.Cleanup(func() { runGHPR = oldPR })
	for _, state := range []string{"OPEN", "CLOSED", "MERGED", "UNKNOWN"} {
		runGHPR = func(context.Context, string, int) ([]byte, error) {
			return []byte(`{"number":33,"url":"https://github.com/gkzz/example/pull/33","state":"` + state + `","headRefOid":"abc","updatedAt":"2026-11-03T12:42:39Z","statusCheckRollup":null}`), nil
		}
		prs, err := fetchPullRequests(context.Background(), "gkzz/example", []dashpkg.Item{{PRLink: "../pull/33"}})
		if state == "UNKNOWN" {
			if err == nil {
				t.Fatal("accepted unknown state")
			}
		} else if err != nil || len(prs) != 1 || prs[0].Checks == nil {
			t.Fatalf("state=%s prs=%+v err=%v", state, prs, err)
		}
	}
	prs, err := fetchPullRequests(context.Background(), "gkzz/example", nil)
	if err != nil || prs == nil || len(prs) != 0 {
		t.Fatalf("prs=%v err=%v", prs, err)
	}
}
