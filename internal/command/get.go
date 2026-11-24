package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	dashpkg "github.com/gkzz/renodash/internal/dashboard"
)

type issueSource struct {
	Repository     string `json:"repository"`
	IssueNumber    int    `json:"issue_number"`
	IssueURL       string `json:"issue_url"`
	FetchedAt      string `json:"fetched_at"`
	IssueUpdatedAt string `json:"issue_updated_at"`
}

type getResult struct {
	Source issueSource `json:"source"`
	dashpkg.Dashboard
	PullRequests *[]pullRequestMetadata `json:"pull_requests,omitempty"`
	Retrieval    *retrievalResult       `json:"retrieval,omitempty"`
}

type githubIssue struct {
	Body      *string `json:"body"`
	Number    int     `json:"number"`
	HTMLURL   string  `json:"html_url"`
	UpdatedAt string  `json:"updated_at"`
}

var getNow = time.Now
var parseGetDashboard = dashpkg.Parse

func executeGet(client githubClient, args []string, stdout, stderr io.Writer) error {
	return executeGetInput(client, args, strings.NewReader(""), stdout, stderr)
}

func executeGetInput(client githubClient, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		_, err := fmt.Fprintln(stdout, "Usage: renodash get [--with-pr-metadata [--allow-partial]] [--stdin | OWNER/REPO#ISSUE_NUMBER]\n\nFetch a GitHub issue with gh and parse its Dashboard body.\nUse --with-pr-metadata to fetch linked PR states and CI checks.\nUse --allow-partial to retain results when individual PRs fail.\nUse --stdin to read one target (at most 4096 bytes) from standard input.")
		return err
	}
	withPR, allowPartial, fromStdin := false, false, false
	var positional []string
	for _, arg := range args {
		if arg == "--with-pr-metadata" {
			withPR = true
		} else if arg == "--allow-partial" {
			allowPartial = true
		} else if arg == "--stdin" {
			fromStdin = true
		} else {
			positional = append(positional, arg)
		}
	}
	if allowPartial && !withPR {
		return errors.New("--allow-partial requires --with-pr-metadata")
	}
	if fromStdin {
		if len(positional) != 0 {
			return errors.New("--stdin cannot be combined with a positional target")
		}
		data, err := io.ReadAll(io.LimitReader(stdin, 4097))
		if err != nil {
			return fmt.Errorf("read target: %w", err)
		}
		if len(data) > 4096 {
			return errors.New("stdin target exceeds 4096 bytes")
		}
		positional = []string{strings.TrimSpace(string(data))}
	}
	args = positional
	if len(args) != 1 {
		return errors.New("get requires exactly one OWNER/REPO#ISSUE_NUMBER argument")
	}
	repo, number, err := parseIssueRef(args[0])
	if err != nil {
		return err
	}
	signals, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(signals, 5*time.Minute)
	defer cancel()
	data, err := fetchIssue(ctx, client, repo, number)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", args[0], err)
	}
	fetchedAt := getNow().UTC().Format(time.RFC3339)
	var issue githubIssue
	if err := json.Unmarshal(data, &issue); err != nil {
		return fmt.Errorf("decode GitHub issue JSON: %w", err)
	}
	canonicalRepo, validURL := issueRepositoryFromURL(issue.HTMLURL, repo, number)
	if issue.Number != number || !validURL || issue.UpdatedAt == "" {
		return errors.New("GitHub issue JSON has missing or mismatched metadata")
	}
	if _, err := time.Parse(time.RFC3339, issue.UpdatedAt); err != nil {
		return fmt.Errorf("invalid issue updated_at: %w", err)
	}
	if issue.Body == nil || strings.TrimSpace(*issue.Body) == "" {
		return errors.New("GitHub issue body is empty")
	}
	dashboard, err := parseGetDashboard(strings.NewReader(*issue.Body))
	if err != nil {
		return fmt.Errorf("parse Dashboard: %w", err)
	}
	if dashboard.Items == nil {
		dashboard.Items = []dashpkg.Item{}
	}
	if dashboard.BulkActions == nil {
		dashboard.BulkActions = []dashpkg.BulkAction{}
	}
	if dashboard.UnknownControls == nil {
		dashboard.UnknownControls = []string{}
	}
	result := getResult{Source: issueSource{canonicalRepo, number, issue.HTMLURL, fetchedAt, issue.UpdatedAt}, Dashboard: dashboard}
	if withPR {
		prs, retrieval, err := collectPullRequests(ctx, client, canonicalRepo, dashboard.Items, allowPartial, defaultRetryPolicy())
		if err != nil {
			return err
		}
		result.PullRequests = &prs
		if allowPartial {
			result.Retrieval = &retrieval
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("encode JSON: %w", err)
	}
	for _, control := range dashboard.UnknownControls {
		if _, err := fmt.Fprintf(stderr, "warning: unsupported Renovate control: %q\n", control); err != nil {
			return fmt.Errorf("write warning: %w", err)
		}
	}
	if _, err := stdout.Write(output.Bytes()); err != nil {
		return fmt.Errorf("write JSON: %w", err)
	}
	return nil
}

func matchesIssueURL(rawURL, repo string, number int) bool {
	_, ok := issueRepositoryFromURL(rawURL, repo, number)
	return ok
}

func issueRepositoryFromURL(rawURL, repo string, number int) (string, bool) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" || !strings.EqualFold(u.Host, "github.com") || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" {
		return "", false
	}
	pathRepo, ok := strings.CutSuffix(u.EscapedPath(), fmt.Sprintf("/issues/%d", number))
	if !ok || !strings.EqualFold(pathRepo, "/"+repo) {
		return "", false
	}
	return strings.TrimPrefix(pathRepo, "/"), true
}

// fetchIssue owns the issue request budget, independently of later PR requests.
func fetchIssue(parent context.Context, client githubClient, repo string, number int) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	data, err := client.FetchIssue(ctx, repo, number)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return data, nil
}
