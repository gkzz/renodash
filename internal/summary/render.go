// Package summary renders saved offline observations as an Actions Summary.
package summary

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

type offlineResult struct {
	Source *struct {
		IssueNumber int    `json:"issue_number"`
		IssueURL    string `json:"issue_url"`
		FetchedAt   string `json:"fetched_at"`
	} `json:"source"`
	Items           []struct{ Section, Action, Title string } `json:"items"`
	BulkActions     []json.RawMessage                         `json:"bulk_actions"`
	UnknownControls []json.RawMessage                         `json:"unknown_controls"`
	PullRequests    []struct {
		Number  int    `json:"number"`
		URL     string `json:"url"`
		State   string `json:"state"`
		IsDraft bool   `json:"is_draft"`
		Title   string `json:"title"`
		Checks  []struct {
			Name, Context             *string
			Conclusion, Status, State string
		} `json:"checks"`
	} `json:"pull_requests"`
}

var newlines = regexp.MustCompile(`[\r\n]+`)
var escape = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `\`, "&#92;", "*", "&#42;", "_", "&#95;", "[", "&#91;", "]", "&#93;", "|", "&#124;", "`", "&#96;")

func safe(s string) string { return newlines.ReplaceAllString(escape.Replace(s), " ") }

// linkURL accepts absolute HTTP(S) links and encodes Markdown delimiters.
// Reject control characters before parsing: url.Parse can otherwise strip them.
func linkURL(raw string) (string, error) {
	if strings.ContainsFunc(raw, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) {
		return "", fmt.Errorf("URL contains whitespace or control characters")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || u.User != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return "", fmt.Errorf("expected an absolute HTTP(S) URL")
	}
	return strings.NewReplacer("(", "%28", ")", "%29", "<", "%3C", ">", "%3E", `\`, "%5C", "|", "%7C", "[", "%5B", "]", "%5D", "`", "%60").Replace(u.String()), nil
}

// RenderOffline converts one renodash JSON object into offline PoC Markdown.
// It validates the entire input before writing; output errors are returned.
func RenderOffline(r io.Reader, w io.Writer) error {
	var result *offlineResult
	dec := json.NewDecoder(r)
	if err := dec.Decode(&result); err != nil {
		return fmt.Errorf("decode offline summary: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("offline summary: expected one JSON value")
		}
		return fmt.Errorf("decode trailing input: %w", err)
	}
	if result == nil || result.Source == nil || result.Source.IssueNumber <= 0 || result.Source.FetchedAt == "" {
		return fmt.Errorf("offline summary: missing source metadata")
	}
	issueURL, err := linkURL(result.Source.IssueURL)
	if err != nil {
		return fmt.Errorf("issue URL: %w", err)
	}
	prURLs := make([]string, len(result.PullRequests))
	for i, pr := range result.PullRequests {
		if pr.Number <= 0 {
			return fmt.Errorf("PR %d: missing number", i)
		}
		prURLs[i], err = linkURL(pr.URL)
		if err != nil {
			return fmt.Errorf("PR %d URL: %w", pr.Number, err)
		}
	}
	var b bytes.Buffer
	fmt.Fprintln(&b, "## Renovate Dependency Dashboard (saved GitHub snapshots)")
	fmt.Fprintln(&b, "\nPR states and checks are real GitHub responses captured and saved in advance for offline testing. They reflect capture time, not current GitHub state.")
	fmt.Fprintln(&b, "\nThe Dashboard body is constructed for testing with real PR references; it is not a captured live Dashboard.")
	fmt.Fprintln(&b, "See testdata/pr-metadata/README.md for source URLs, capture time, and provenance.")
	fmt.Fprintf(&b, "\n- **Issue:** [#%d](%s)\n- **Test execution:** %s\n- **Candidates:** %d\n- **Bulk actions:** %d\n- **Unknown controls:** %d\n- **Linked PRs:** %d\n", result.Source.IssueNumber, issueURL, safe(result.Source.FetchedAt), len(result.Items), len(result.BulkActions), len(result.UnknownControls), len(result.PullRequests))
	fmt.Fprintln(&b, "\n### Update candidates\n\n| Section | Action | Title |\n| --- | --- | --- |")
	if len(result.Items) == 0 {
		fmt.Fprintln(&b, "| - | - | No candidates |")
	}
	for i, item := range result.Items {
		if i == 20 {
			break
		}
		fmt.Fprintf(&b, "| %s | %s | %s |\n", safe(item.Section), safe(item.Action), safe(item.Title))
	}
	if len(result.Items) > 20 {
		fmt.Fprintln(&b, "\n_Showing the first 20 candidates. Download the artifact for the full results._")
	}
	fmt.Fprint(&b, "\n### Linked pull requests\n\n")
	if len(result.PullRequests) == 0 {
		fmt.Fprintln(&b, "No linked pull requests were found in the Dependency Dashboard. PR metadata verification was skipped.")
	} else {
		fmt.Fprintln(&b, "| PR | State | Draft | Title | Checks |\n| --- | --- | --- | --- | --- |")
		for i, pr := range result.PullRequests {
			checks := make([]string, 0, len(pr.Checks))
			for _, c := range pr.Checks {
				name := "Unnamed check"
				if c.Name != nil {
					name = *c.Name
				} else if c.Context != nil {
					name = *c.Context
				}
				state := "Unknown"
				for _, value := range []string{c.Conclusion, c.Status, c.State} {
					if value != "" {
						state = value
						break
					}
				}
				checks = append(checks, safe(name)+": "+safe(state))
			}
			checkText := "Unknown (no checks returned)"
			if len(checks) > 0 {
				checkText = strings.Join(checks, "<br>")
			}
			fmt.Fprintf(&b, "| [#%d](%s) | %s | %t | %s | %s |\n", pr.Number, prURLs[i], safe(pr.State), pr.IsDraft, safe(pr.Title), checkText)
		}
		fmt.Fprintln(&b, "\nCheck results are observations, not merge approval.")
	}
	n, err := w.Write(b.Bytes())
	if err == nil && n != b.Len() {
		err = io.ErrShortWrite
	}
	if err != nil {
		return fmt.Errorf("write offline summary: %w", err)
	}
	return nil
}
