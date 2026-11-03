package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gkzz/renodash"
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
	renodash.Dashboard
}

type githubIssue struct {
	Body      *string `json:"body"`
	Number    int     `json:"number"`
	HTMLURL   string  `json:"html_url"`
	UpdatedAt string  `json:"updated_at"`
}

var getNow = time.Now
var parseGetDashboard = renodash.Parse

var runGH = func(ctx context.Context, endpoint string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "gh", "api", endpoint)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
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
			return nil, fmt.Errorf("gh api failed: %w", err)
		}
	}
	return output, nil
}

func parseIssueRef(value string) (string, int, error) {
	repo, numberText, ok := strings.Cut(value, "#")
	parts := strings.Split(repo, "/")
	if !ok || len(parts) != 2 || !validRepoPart(parts[0]) || !validRepoPart(parts[1]) || numberText == "" {
		return "", 0, errors.New("invalid issue reference: expected OWNER/REPO#ISSUE_NUMBER")
	}
	for _, c := range numberText {
		if c < '0' || c > '9' {
			return "", 0, errors.New("invalid issue reference: issue number must be a positive integer")
		}
	}
	number, err := strconv.Atoi(numberText)
	if err != nil || number <= 0 {
		return "", 0, errors.New("invalid issue reference: issue number must be a positive integer")
	}
	return repo, number, nil
}

func validRepoPart(part string) bool {
	if part == "" || part == "." || part == ".." {
		return false
	}
	for _, c := range part {
		if !(unicode.IsLetter(c) && c < 128 || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.') {
			return false
		}
	}
	return true
}

func executeGet(args []string, stdout, stderr io.Writer) error {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		_, err := fmt.Fprintln(stdout, "Usage: renodash get OWNER/REPO#ISSUE_NUMBER\n\nFetch a GitHub issue with gh and parse its Dashboard body.")
		return err
	}
	if len(args) != 1 {
		return errors.New("get requires exactly one OWNER/REPO#ISSUE_NUMBER argument")
	}
	repo, number, err := parseIssueRef(args[0])
	if err != nil {
		return err
	}
	endpoint := fmt.Sprintf("repos/%s/issues/%d", repo, number)
	data, err := runGH(context.Background(), endpoint)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", args[0], err)
	}
	fetchedAt := getNow().UTC().Format(time.RFC3339)
	var issue githubIssue
	if err := json.Unmarshal(data, &issue); err != nil {
		return fmt.Errorf("decode GitHub issue JSON: %w", err)
	}
	if issue.Number != number || issue.HTMLURL != fmt.Sprintf("https://github.com/%s/issues/%d", repo, number) || issue.UpdatedAt == "" {
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
		dashboard.Items = []renodash.Item{}
	}
	if dashboard.BulkActions == nil {
		dashboard.BulkActions = []renodash.BulkAction{}
	}
	if dashboard.UnknownControls == nil {
		dashboard.UnknownControls = []string{}
	}
	result := getResult{Source: issueSource{repo, number, issue.HTMLURL, fetchedAt, issue.UpdatedAt}, Dashboard: dashboard}
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("encode JSON: %w", err)
	}
	if _, err := stdout.Write(output.Bytes()); err != nil {
		return fmt.Errorf("write JSON: %w", err)
	}
	for _, control := range dashboard.UnknownControls {
		if _, err := fmt.Fprintf(stderr, "warning: unsupported Renovate control: %s\n", control); err != nil {
			return fmt.Errorf("write warning: %w", err)
		}
	}
	return nil
}
