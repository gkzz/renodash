package command

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

const issueAuthError = "GitHub authentication failed; set GH_TOKEN or GITHUB_TOKEN, or run gh auth login"

type issueHTTPClient struct {
	ghClient
	http  *http.Client
	token string
}

func newIssueHTTPClient() *issueHTTPClient {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	return &issueHTTPClient{http: &http.Client{
		Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}}
}

func (c *issueHTTPClient) FetchIssue(ctx context.Context, repo string, number int) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c.token == "" {
		token, err := issueToken(ctx)
		if err != nil {
			return nil, err
		}
		c.token = token
	}

	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/issues/%d", repo, number)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, errors.New("create GitHub issue request")
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")

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

func issueToken(ctx context.Context) (string, error) {
	for _, name := range []string{"GH_TOKEN", "GITHUB_TOKEN"} {
		if token := strings.TrimSpace(os.Getenv(name)); token != "" {
			return token, nil
		}
	}
	// Only stdout contains the credential. Discard all subprocess diagnostics.
	cmd := exec.CommandContext(ctx, "gh", "auth", "token", "--hostname", "github.com")
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
