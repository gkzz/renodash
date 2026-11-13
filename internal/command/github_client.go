package command

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// githubClient provides the GitHub operations used by the get command.
// Callers own request deadlines and response validation.
type githubClient interface {
	FetchIssue(context.Context, string, int) ([]byte, error)
	FetchPR(context.Context, string, int) ([]byte, error)
}

// ghClient invokes the authenticated GitHub CLI without retaining request state.
type ghClient struct{}

func (ghClient) FetchIssue(ctx context.Context, repo string, number int) ([]byte, error) {
	// https://docs.github.com/en/rest/issues/issues#get-an-issue
	return executeGH(ctx, "api", fmt.Sprintf("repos/%s/issues/%d", repo, number))
}

func (ghClient) FetchPR(ctx context.Context, repo string, number int) ([]byte, error) {
	return executeGH(ctx, "pr", "view", strconv.Itoa(number), "--repo", repo,
		"--json", "number,url,state,title,isDraft,headRefOid,updatedAt,statusCheckRollup")
}

func executeGH(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "gh", args...)
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
			return nil, errors.New("gh authentication failed; run gh auth login")
		case strings.Contains(message, "http 404"), strings.Contains(message, "not found"):
			return nil, errors.New("issue not found or inaccessible")
		default:
			// gh diagnostics may include secrets supplied by the environment or API.
			return nil, fmt.Errorf("gh command failed: %w", err)
		}
	}
	return output, nil
}
