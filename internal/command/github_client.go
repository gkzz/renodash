package command

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// githubClient provides the GitHub operations used by the get command.
// Callers own request deadlines and response validation.
type githubClient interface {
	FetchIssue(context.Context, string, int) ([]byte, error)
	FetchPR(context.Context, string, int) ([]byte, error)
}

// ghClient fetches PR metadata through the authenticated GitHub CLI.
type ghClient struct {
	token string
}

// Keep stdout internally on failure so structured HTTP headers can be read.
func (c ghClient) runGH(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "gh", args...)
	if c.token != "" {
		cmd.Env = append(cmd.Environ(), "GH_TOKEN="+c.token, "GITHUB_TOKEN=")
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if errors.Is(err, exec.ErrNotFound) {
			return nil, errors.New("gh is not installed or not in PATH")
		}
		message := strings.ToLower(stderr.String())
		switch {
		case strings.Contains(message, "authentication"), strings.Contains(message, "not logged in"), strings.Contains(message, "http 401"):
			return output, errors.New("gh authentication failed; run gh auth login")
		case strings.Contains(message, "http 404"), strings.Contains(message, "not found"):
			return output, errors.New("issue not found or inaccessible")
		default:
			// gh diagnostics may include secrets supplied by the environment or API.
			return output, fmt.Errorf("gh command failed: %w", err)
		}
	}
	return output, nil
}
