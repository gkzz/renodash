package command

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	dashpkg "github.com/gkzz/renodash/internal/dashboard"
)

// Keep GitHub's check fields unchanged: CheckRun and StatusContext use different
// fields, and neither an empty list nor a pending check implies merge approval.
type pullRequestMetadata struct {
	Mergeable        string            `json:"mergeable"`
	MergeStateStatus string            `json:"merge_state_status"`
	FetchedAt        string            `json:"fetched_at"`
	Number           int               `json:"number"`
	URL              string            `json:"url"`
	State            string            `json:"state"`
	Title            string            `json:"title"`
	IsDraft          bool              `json:"is_draft"`
	HeadRefOID       string            `json:"head_sha"`
	UpdatedAt        string            `json:"updated_at"`
	Checks           []json.RawMessage `json:"checks"`
}

// Resolve only GitHub PR links in the Dashboard's repository. Never pass a
// Dashboard-supplied host or URL as an API destination.
func linkedPRNumber(link, repo string) (int, error) {
	path := link
	if strings.HasPrefix(path, "../pull/") {
		path = "/" + repo + strings.TrimPrefix(path, "..")
	} else if strings.HasPrefix(path, "pull/") {
		path = "/" + repo + "/" + path
	}
	if strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "//") {
		path = "https://github.com" + path
	}
	u, err := url.Parse(path)
	if err != nil || u.Scheme != "https" || !strings.EqualFold(u.Host, "github.com") ||
		u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return 0, fmt.Errorf("unsupported PR link %q", link)
	}
	prefix, numberText, ok := strings.Cut(u.EscapedPath(), "/pull/")
	if !ok || !strings.EqualFold(prefix, "/"+repo) || numberText == "" {
		return 0, fmt.Errorf("unsupported PR link %q", link)
	}
	for _, c := range numberText {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid PR number in %q", link)
		}
	}
	number, err := strconv.Atoi(numberText)
	if err != nil || number <= 0 {
		return 0, fmt.Errorf("invalid PR number in %q", link)
	}
	return number, nil
}

func fetchPullRequests(ctx context.Context, client githubClient, repo string, items []dashpkg.Item) ([]pullRequestMetadata, error) {
	prs, _, err := collectPullRequests(ctx, client, repo, items, false, defaultRetryPolicy())
	return prs, err
}

var errPRDecode = errors.New("invalid PR response JSON")

func decodePullRequest(data []byte, repo string, number int, fetchedAt string) (pullRequestMetadata, error) {
	var pr struct {
		Mergeable        string            `json:"mergeable"`
		MergeStateStatus string            `json:"mergeStateStatus"`
		Number           int               `json:"number"`
		URL              string            `json:"url"`
		State            string            `json:"state"`
		Title            string            `json:"title"`
		IsDraft          bool              `json:"isDraft"`
		HeadRefOID       string            `json:"headRefOid"`
		UpdatedAt        string            `json:"updatedAt"`
		Checks           []json.RawMessage `json:"statusCheckRollup"`
	}
	if err := json.Unmarshal(data, &pr); err != nil {
		return pullRequestMetadata{}, fmt.Errorf("decode PR %d: %w", number, errors.Join(errPRDecode, err))
	}
	urlNumber, urlErr := linkedPRNumber(pr.URL, repo)
	if pr.Number != number || urlErr != nil || urlNumber != number ||
		(pr.State != "OPEN" && pr.State != "CLOSED" && pr.State != "MERGED") ||
		pr.HeadRefOID == "" || pr.UpdatedAt == "" {
		return pullRequestMetadata{}, fmt.Errorf("PR %d JSON has missing or mismatched metadata", number)
	}
	if pr.Checks == nil {
		pr.Checks = []json.RawMessage{}
	}
	if pr.Mergeable == "" {
		pr.Mergeable = "UNKNOWN"
	}
	if pr.MergeStateStatus == "" {
		pr.MergeStateStatus = "UNKNOWN"
	}
	return pullRequestMetadata{
		Number: pr.Number, URL: pr.URL, State: pr.State, Title: pr.Title,
		IsDraft: pr.IsDraft, HeadRefOID: pr.HeadRefOID, UpdatedAt: pr.UpdatedAt,
		Checks: pr.Checks, Mergeable: pr.Mergeable, MergeStateStatus: pr.MergeStateStatus, FetchedAt: fetchedAt,
	}, nil
}

// Give each unique PR its own budget while preserving parent cancellation.
func fetchPR(parent context.Context, client githubClient, repo string, number int) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	data, err := client.FetchPR(ctx, repo, number)
	if err := parent.Err(); err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return data, nil
}
