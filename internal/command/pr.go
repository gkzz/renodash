package command

import (
	"context"
	"encoding/json"
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
	Number     int               `json:"number"`
	URL        string            `json:"url"`
	State      string            `json:"state"`
	Title      string            `json:"title"`
	IsDraft    bool              `json:"is_draft"`
	HeadRefOID string            `json:"head_sha"`
	UpdatedAt  string            `json:"updated_at"`
	Checks     []json.RawMessage `json:"checks"`
}

var runGHPR = func(ctx context.Context, repo string, number int) ([]byte, error) {
	return executeGH(ctx, "pr", "view", strconv.Itoa(number), "--repo", repo,
		"--json", "number,url,state,title,isDraft,headRefOid,updatedAt,statusCheckRollup")
}

// Resolve only GitHub PR links in the Dashboard's repository. Never pass a
// Dashboard-supplied host or arbitrary URL to gh.
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

func fetchPullRequests(ctx context.Context, repo string, items []dashpkg.Item) ([]pullRequestMetadata, error) {
	result := []pullRequestMetadata{}
	seen := map[int]bool{}
	for _, item := range items {
		if item.PRLink == "" {
			continue
		}
		number, err := linkedPRNumber(item.PRLink, repo)
		if err != nil {
			return nil, err
		}
		if seen[number] {
			continue
		}
		seen[number] = true
		data, err := fetchPR(ctx, repo, number)
		if err != nil {
			return nil, fmt.Errorf("fetch PR %s#%d: %w", repo, number, err)
		}
		var pr struct {
			Number     int               `json:"number"`
			URL        string            `json:"url"`
			State      string            `json:"state"`
			Title      string            `json:"title"`
			IsDraft    bool              `json:"isDraft"`
			HeadRefOID string            `json:"headRefOid"`
			UpdatedAt  string            `json:"updatedAt"`
			Checks     []json.RawMessage `json:"statusCheckRollup"`
		}
		if err := json.Unmarshal(data, &pr); err != nil {
			return nil, fmt.Errorf("decode PR %d: %w", number, err)
		}
		urlNumber, urlErr := linkedPRNumber(pr.URL, repo)
		if pr.Number != number || urlErr != nil || urlNumber != number ||
			(pr.State != "OPEN" && pr.State != "CLOSED" && pr.State != "MERGED") ||
			pr.HeadRefOID == "" || pr.UpdatedAt == "" {
			return nil, fmt.Errorf("PR %d JSON has missing or mismatched metadata", number)
		}
		if pr.Checks == nil {
			pr.Checks = []json.RawMessage{}
		}
		result = append(result, pullRequestMetadata{pr.Number, pr.URL, pr.State, pr.Title, pr.IsDraft, pr.HeadRefOID, pr.UpdatedAt, pr.Checks})
	}
	return result, nil
}

// Give each unique PR its own budget while preserving parent cancellation.
func fetchPR(parent context.Context, repo string, number int) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	data, err := runGHPR(ctx, repo, number)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return data, nil
}
