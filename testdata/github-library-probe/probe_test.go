package probe

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/auth"
	"github.com/google/go-github/v93/github"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGoGH(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	t.Setenv("GH_CONFIG_DIR", dir)
	t.Setenv("GH_PATH", "")
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	if token, _ := auth.TokenForHost("github.com"); token != "" {
		t.Fatal("unexpected token without gh")
	}
	fake := filepath.Join(dir, "gh")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nif [ \"$1 $2 $3 $4 $5\" != 'auth token --secure-storage --hostname github.com' ]; then exit 1; fi\nprintf fake-keyring\n"), 0700); err != nil {
		t.Fatal(err)
	}
	if token, _ := auth.TokenForHost("github.com"); token != "fake-keyring" {
		t.Fatal("keyring path did not call fake gh")
	}
	t.Setenv("PATH", t.TempDir())
	calls := 0
	client, err := api.NewGraphQLClient(api.ClientOptions{Host: "github.com", AuthToken: "fake-explicit", LogIgnoreEnv: true, Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Host != "api.github.com" || r.Header.Get("Authorization") != "token fake-explicit" {
			t.Error("unexpected origin or authorization")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Retry-After": []string{"7"}}, Body: io.NopCloser(strings.NewReader(`{"data":{"viewer":null},"errors":[{"type":"RATE_LIMITED","message":"synthetic-marker"}]}`))}, nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	var result any
	err = client.DoWithContext(context.Background(), "query { viewer { login } }", nil, &result)
	var graphqlErr *api.GraphQLError
	if calls != 1 || !errors.As(err, &graphqlErr) || !strings.Contains(graphqlErr.Error(), "synthetic-marker") {
		t.Fatalf("call count or error text differed: %d", calls)
	}
	t.Logf("no gh: missing credential; fake gh: secure-storage lookup; explicit token: 1 request; GraphQL error exposes upstream message unless sanitized (%T)", err)
}

func TestGoGitHub(t *testing.T) {
	calls := 0
	tr := roundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Host != "api.github.com" || r.Header.Get("Authorization") != "Bearer fake-explicit" {
			t.Error("unexpected origin or authorization")
		}
		return &http.Response{StatusCode: 403, Request: r, Header: http.Header{"X-Ratelimit-Remaining": []string{"42"}, "Retry-After": []string{"7"}}, Body: io.NopCloser(strings.NewReader(`{"message":"You have exceeded a secondary rate limit. synthetic-marker","documentation_url":"https://docs.github.com/rest/using-the-rest-api/rate-limits-for-the-rest-api#about-secondary-rate-limits"}`))}, nil
	})
	c, err := github.NewClient(github.WithAuthToken("fake-explicit"), github.WithTransport(tr))
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		_, _, err = c.Issues.Get(context.Background(), "owner", "repo", 1)
		var limit *github.AbuseRateLimitError
		if !errors.As(err, &limit) {
			t.Fatalf("expected secondary rate-limit error, got %T", err)
		}
		t.Logf("call %d returned %T", calls, err)
	}
	if calls != 1 {
		t.Fatalf("expected one transport call, got %d", calls)
	}
}
