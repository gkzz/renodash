package command

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// githubClient provides the GitHub operations used by the get command.
// Callers own request deadlines and response validation.
type githubClient interface {
	FetchIssue(context.Context, string, int) ([]byte, error)
	FetchPR(context.Context, string, int) ([]byte, error)
}

const githubAuthError = "GitHub authentication failed; set GH_TOKEN or GITHUB_TOKEN, or run gh auth login"

type githubHTTPClient struct {
	http  *http.Client
	mu    sync.Mutex
	token string
}

func newGitHubHTTPClient() *githubHTTPClient {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	// HTTP/2 debug logging can print the Authorization header.
	transport.Protocols = new(http.Protocols)
	transport.Protocols.SetHTTP1(true)
	// Clone can retain the default transport's TLS protocol list.
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = new(tls.Config)
	}
	transport.TLSClientConfig.NextProtos = []string{"http/1.1"}
	return &githubHTTPClient{http: &http.Client{
		Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}}
}

func (c *githubHTTPClient) credentials(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if c.token == "" {
		token, err := githubToken(ctx)
		if err != nil {
			return "", err
		}
		c.token = token
	}
	return c.token, nil
}

func githubToken(ctx context.Context) (string, error) {
	for _, name := range []string{"GH_TOKEN", "GITHUB_TOKEN"} {
		if token := strings.TrimSpace(os.Getenv(name)); token != "" {
			return token, nil
		}
	}
	// Only stdout contains the credential. Discard all subprocess diagnostics.
	cmd := exec.CommandContext(ctx, "gh", "auth", "token", "--hostname", "github.com")
	// Any environment tokens reaching this branch were empty after trimming.
	cmd.Env = append(cmd.Environ(), "GH_TOKEN=", "GITHUB_TOKEN=")
	cmd.Stderr = io.Discard
	output, err := cmd.Output()
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	token := strings.TrimSpace(string(output))
	if err != nil || token == "" {
		return "", errors.New(githubAuthError)
	}
	return token, nil
}
