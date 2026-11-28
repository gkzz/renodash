package command

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
)

func (c *githubHTTPClient) FetchIssue(ctx context.Context, repo string, number int) ([]byte, error) {
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
		return nil, errors.New(githubAuthError)
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
