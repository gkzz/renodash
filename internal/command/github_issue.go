package command

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
)

const issueAuthError = "GitHub authentication failed; set GH_TOKEN or GITHUB_TOKEN, or run gh auth login"

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

func (c *githubHTTPClient) FetchIssue(ctx context.Context, repo string, number int) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	token, err := c.credentials(ctx)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/issues/%d", repo, number)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, errors.New("create GitHub issue request")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "renodash")

	resp, err := c.http.Do(req)
	if ctx.Err() != nil {
		if resp != nil {
			resp.Body.Close()
		}
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, errors.New("GitHub issue request failed")
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return nil, errors.New(issueAuthError)
	case http.StatusNotFound:
		return nil, errors.New("issue not found or inaccessible")
	default:
		return nil, fmt.Errorf("GitHub issue request failed (HTTP %d)", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, errors.New("read GitHub issue response")
	}
	return data, nil
}

func (c *githubHTTPClient) credentials(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if c.token == "" {
		token, err := issueToken(ctx)
		if err != nil {
			return "", err
		}
		c.token = token
	}
	return c.token, nil
}

func issueToken(ctx context.Context) (string, error) {
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
		return "", errors.New(issueAuthError)
	}
	return token, nil
}
