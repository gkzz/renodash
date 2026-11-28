package command

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

type issueRoundTripper func(*http.Request) (*http.Response, error)

func (f issueRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// Use a local TLS server while checking the original production URL.
func useIssueServer(t *testing.T, handler http.HandlerFunc) *issueHTTPClient {
	t.Helper()
	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)
	localURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	client := newIssueHTTPClient()
	client.http.Transport = issueRoundTripper(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.String() != "https://api.github.com/repos/Owner/Repo/issues/4" {
			return nil, errors.New("unexpected issue endpoint")
		}
		local := req.Clone(req.Context())
		local.URL.Host = localURL.Host
		return server.Client().Transport.RoundTrip(local)
	})
	return client
}

func routeIssueServer(t *testing.T, server *httptest.Server) *http.Transport {
	t.Helper()
	// Route the Run entry point to the local TLS fixture.
	// Call only from tests that run serially.
	original := http.DefaultTransport
	transport := original.(*http.Transport).Clone()
	transport.Proxy = func(*http.Request) (*url.URL, error) {
		t.Error("Issue request consulted the proxy")
		return nil, errors.New("unexpected proxy lookup")
	}
	transport.Protocols = new(http.Protocols)
	transport.Protocols.SetHTTP1(true)
	transport.Protocols.SetHTTP2(true)
	transport.TLSClientConfig = server.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
	transport.TLSClientConfig.ServerName = "example.com"
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		if addr != "api.github.com:443" {
			t.Error("unexpected dial address")
			return nil, errors.New("unexpected dial address")
		}
		return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
	}
	http.DefaultTransport = transport
	t.Cleanup(func() { http.DefaultTransport = original })

	return transport
}

func TestRunFetchesIssueOverHTTP1(t *testing.T) {
	t.Setenv("GH_TOKEN", "fixture-token")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	t.Setenv("NO_PROXY", "")

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.ProtoMajor != 1 {
			t.Errorf("issue request negotiated %s", req.Proto)
		}
		if req.Method != http.MethodGet || req.Host != "api.github.com" || req.URL.Path != "/repos/Owner/Repo/issues/4" {
			t.Error("unexpected issue request")
		}
		if req.Header.Get("Authorization") != "Bearer fixture-token" {
			t.Error("unexpected authorization header")
		}
		if req.UserAgent() != "renodash" || req.Header.Get("X-GitHub-Api-Version") != "2022-11-28" {
			t.Error("unexpected GitHub API headers")
		}
		fmt.Fprint(w, `{"body":"## Open","number":4,"html_url":"https://github.com/Owner/Repo/issues/4","updated_at":"2026-11-03T12:42:39Z"}`)
	}))
	server.EnableHTTP2 = true
	server.TLS = &tls.Config{NextProtos: []string{"h2", "http/1.1"}}
	server.StartTLS()
	t.Cleanup(server.Close)

	transport := routeIssueServer(t, server)

	var out, diagnostic bytes.Buffer
	code := Run([]string{"get", "Owner/Repo#4"}, strings.NewReader(""), &out, &diagnostic)
	if code != 0 || diagnostic.Len() != 0 {
		t.Fatalf("code=%d stderr=%q", code, diagnostic.String())
	}
	var result getResult
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Source.Repository != "Owner/Repo" || result.Source.IssueNumber != 4 ||
		result.Source.IssueURL != "https://github.com/Owner/Repo/issues/4" || result.PullRequests != nil {
		t.Fatalf("unexpected result: %+v", result)
	}
	t.Run("reject untrusted certificate", func(t *testing.T) {
		untrusted := transport.Clone()
		untrusted.TLSClientConfig.RootCAs = x509.NewCertPool()
		http.DefaultTransport = untrusted
		t.Cleanup(func() { http.DefaultTransport = transport })
		var out, diagnostic bytes.Buffer
		code := Run([]string{"get", "Owner/Repo#4"}, nil, &out, &diagnostic)
		if code != 1 || out.Len() != 0 || diagnostic.String() != "error: fetch Owner/Repo#4: GitHub issue request failed\n" {
			t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), diagnostic.String())
		}
	})
}

func TestFetchIssueCredentials(t *testing.T) {
	const issue = `{"body":"## Open","number":4,"html_url":"https://github.com/Owner/Repo/issues/4","updated_at":"2026-11-03T12:42:39Z"}`
	for _, tc := range []struct {
		name, gh, github, want string
	}{
		{"GH_TOKEN first", " primary ", "secondary", "primary"},
		{"GITHUB_TOKEN fallback", " ", "secondary", "secondary"},
		{"saved gh login", "", "", "saved"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GH_TOKEN", tc.gh)
			t.Setenv("GITHUB_TOKEN", tc.github)
			t.Setenv("PATH", t.TempDir())
			if tc.name == "saved gh login" {
				installFakeGH(t, "auth-token", "saved\r\n")
			}
			client := useIssueServer(t, func(w http.ResponseWriter, req *http.Request) {
				if req.Header.Get("Authorization") != "Bearer "+tc.want ||
					req.Header.Get("Accept") != "application/vnd.github+json" {
					t.Error("unexpected request headers")
				}
				fmt.Fprint(w, issue)
			})
			var out, diagnostic bytes.Buffer
			code := runWithClient(client, []string{"get", "Owner/Repo#04"}, nil, &out, &diagnostic)
			if code != 0 || diagnostic.Len() != 0 || !strings.Contains(out.String(), `"issue_number": 4`) ||
				!strings.Contains(out.String(), `"repository": "Owner/Repo"`) {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), diagnostic.String())
			}
			t.Setenv("GH_TOKEN", "changed")
			t.Setenv("PATH", t.TempDir())
			data, err := client.FetchIssue(context.Background(), "Owner/Repo", 4)
			if string(data) != issue || err != nil {
				t.Fatalf("response=%q error=%v", data, err)
			}
		})
	}
}

func TestFetchIssueHTTPErrors(t *testing.T) {
	t.Setenv("GH_TOKEN", "credential-marker")
	t.Setenv("GITHUB_TOKEN", "fallback-marker")
	t.Setenv("PATH", t.TempDir())
	for _, tc := range []struct {
		status int
		want   string
	}{
		{302, "GitHub issue request failed (HTTP 302)"},
		{401, "GitHub authentication failed; set GH_TOKEN or GITHUB_TOKEN, or run gh auth login"},
		{403, "GitHub issue request failed (HTTP 403)"},
		{404, "issue not found or inaccessible"},
		{429, "GitHub issue request failed (HTTP 429)"},
		{503, "GitHub issue request failed (HTTP 503)"},
	} {
		t.Run(fmt.Sprint(tc.status), func(t *testing.T) {
			client := useIssueServer(t, func(w http.ResponseWriter, req *http.Request) {
				if req.Header.Get("Authorization") != "Bearer credential-marker" {
					t.Error("unexpected credential fallback")
				}
				w.Header().Set("Location", "https://example.invalid/credential-marker")
				w.Header().Set("Content-Length", "200")
				w.WriteHeader(tc.status)
				fmt.Fprint(w, "credential-marker raw-body-marker")
			})
			var out, diagnostic bytes.Buffer
			code := runWithClient(client, []string{"get", "--with-pr-metadata", "--allow-partial", "Owner/Repo#4"}, nil, &out, &diagnostic)
			if code != 1 || out.Len() != 0 || diagnostic.String() != "error: fetch Owner/Repo#4: "+tc.want+"\n" {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), diagnostic.String())
			}
		})
	}
}

func TestFetchIssueTransportErrorIsSanitized(t *testing.T) {
	t.Setenv("GH_TOKEN", "credential-marker")
	c := newIssueHTTPClient()
	c.http.Transport = issueRoundTripper(func(req *http.Request) (*http.Response, error) {
		return nil, errors.New(req.Header.Get("Authorization"))
	})
	_, err := c.FetchIssue(context.Background(), "Owner/Repo", 4)
	if err == nil || err.Error() != "GitHub issue request failed" {
		t.Fatalf("error=%v", err)
	}
}

func TestFetchIssueContext(t *testing.T) {
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("PATH", t.TempDir())
	c := newIssueHTTPClient()
	c.http.Transport = issueRoundTripper(func(*http.Request) (*http.Response, error) {
		t.Fatal("request started after cancellation")
		return nil, nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.FetchIssue(ctx, "Owner/Repo", 4); !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}

	t.Setenv("GH_TOKEN", "fixture")
	t.Run("cancel during body read", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		client := useIssueServer(t, func(w http.ResponseWriter, req *http.Request) {
			fmt.Fprint(w, "{")
			w.(http.Flusher).Flush()
			cancel()
			<-req.Context().Done()
		})
		data, err := client.FetchIssue(ctx, "Owner/Repo", 4)
		if data != nil || !errors.Is(err, context.Canceled) {
			t.Fatalf("response=%q error=%v", data, err)
		}
	})
	t.Run("deadline during request", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			client := newIssueHTTPClient()
			client.http.Transport = issueRoundTripper(func(req *http.Request) (*http.Response, error) {
				<-req.Context().Done()
				return nil, req.Context().Err()
			})
			data, err := client.FetchIssue(ctx, "Owner/Repo", 4)
			if data != nil || !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("response=%q error=%v", data, err)
			}
		})
	})
}

func TestRunSkipsIssueAuthentication(t *testing.T) {
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("PATH", t.TempDir())
	for _, tc := range []struct {
		args []string
		code int
	}{
		{[]string{"parse"}, 0},
		{[]string{"version"}, 0},
		{[]string{"--help"}, 0},
		{[]string{"get", "--help"}, 0},
		{[]string{"get", "invalid"}, 1},
	} {
		var diagnostic bytes.Buffer
		code := Run(tc.args, strings.NewReader("## Open"), io.Discard, &diagnostic)
		if code != tc.code || strings.Contains(diagnostic.String(), "authentication") {
			t.Fatalf("%v: code=%d stderr=%q", tc.args, code, diagnostic.String())
		}
	}
}
