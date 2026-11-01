package renodash

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
)

var (
	headingPattern       = regexp.MustCompile(`^ {0,3}(#{1,2})[ \t]+(.+?)[ \t]*$`)
	closingHashesPattern = regexp.MustCompile(`[ \t]+#+[ \t]*$`)
	commentStartPattern  = regexp.MustCompile(`^ {0,3}<!--`)
	fencePattern         = regexp.MustCompile("^ {0,3}(`{3,}|~{3,})(.*)$")
	checkboxPattern      = regexp.MustCompile(`^ {0,3}- \[([ xX])\][ \t]+<!--[ \t]*(.*?)[ \t]*-->[ \t]*(.*)$`)
	branchPattern        = regexp.MustCompile(`^(approve|unschedule|unpend)-branch=([^\s<>]+)$`)
)

// Parse extracts supported checkbox controls from known Renovate operation
// sections, excluding HTML comment blocks, fenced code, and lines indented by
// four or more spaces or tabs. Unrecognized comment bodies are retained in
// UnknownControls. Unknown sections are ignored. Titles retain Markdown.
// On a read error, Parse returns an empty Dashboard and the error.
func Parse(r io.Reader) (Dashboard, error) {
	var dashboard Dashboard
	var section string
	var fence string
	var inComment bool
	reader := bufio.NewReader(r)
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return Dashboard{}, fmt.Errorf("read dashboard: %w", err)
		}
		line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		if inComment {
			// The first closing delimiter ends an HTML comment; comments do not nest.
			inComment = !strings.Contains(line, "-->")
		} else if marker := fencePattern.FindStringSubmatch(line); marker != nil &&
			(fence != "" || marker[1][0] != '`' || !strings.Contains(marker[2], "`")) {
			if fence == "" {
				fence = marker[1]
			} else if marker[1][0] == fence[0] && len(marker[1]) >= len(fence) && strings.TrimSpace(marker[2]) == "" {
				fence = ""
			}
		} else if fence != "" {
			// Headings and controls inside a fence are examples, not operations.
		} else if opening := commentStartPattern.FindStringIndex(line); opening != nil {
			inComment = !strings.Contains(line[opening[1]:], "-->")
		} else if heading := headingPattern.FindStringSubmatch(line); heading != nil {
			section = ""
			if heading[1] == "##" {
				section = strings.TrimSpace(closingHashesPattern.ReplaceAllString(heading[2], ""))
			}
		} else if isOperationSection(section) {
			if checkbox := checkboxPattern.FindStringSubmatch(line); checkbox != nil {
				checked := checkbox[1] != " "
				control := strings.TrimSpace(checkbox[2])
				if control == "approve-all-pending-prs" {
					dashboard.BulkActions = append(dashboard.BulkActions, BulkAction{Action: control, Checked: checked})
				} else if branch := branchPattern.FindStringSubmatch(control); branch != nil {
					dashboard.Items = append(dashboard.Items, Item{
						Action: branch[1], Branch: branch[2], Title: strings.TrimSpace(checkbox[3]),
						Section: section, Checked: checked,
					})
				} else {
					dashboard.UnknownControls = append(dashboard.UnknownControls, control)
				}
			}
		}
		if err == io.EOF {
			return dashboard, nil
		}
	}
}

// isOperationSection limits extraction to sections generated for Dashboard controls.
// See lib/workers/repository/dependency-dashboard.ts in renovatebot/renovate.
func isOperationSection(section string) bool {
	switch section {
	case "Pending Approval", "Awaiting Schedule", "Pending Status Checks",
		"Group Size Not Met", "Rate-Limited", "Errored", "PR Creation Approval Required",
		"PR Edited (Blocked)", "Pending Branch Automerge", "Other Branches", "Open",
		"PR Closed (Blocked)", "Config Migration Needed", "Config Migration Needed (Blocked)":
		return true
	default:
		return false
	}
}
