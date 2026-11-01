package renodash

import (
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestParseFixtures(t *testing.T) {
	tests := []struct {
		name string
		want Dashboard
	}{
		{"dotfiles", Dashboard{Items: []Item{
			{"approve", "renovate/toolchain", "chore(deps): update dependency pnpm to v12.14.0", "Pending Approval", false},
			{"approve", "renovate/major-github-actions-major", "chore(deps): update dependency ubuntu to v26", "Pending Approval", false},
			{"approve", "renovate/lock-file-maintenance-toolchain", "chore(deps): lock file maintenance", "Pending Approval", false},
			{"unpend", "renovate/major-toolchain", "chore(deps): update node.js to v26", "Pending Status Checks", false},
		}, BulkActions: []BulkAction{{"approve-all-pending-prs", false}}}},
		{"maestro-playground", Dashboard{Items: []Item{
			{"approve", "renovate/major-github-actions-major", "chore(deps): update dependency ubuntu to v26", "Pending Approval", false},
			{"unschedule", "renovate/lock-file-maintenance-toolchain", "chore(deps): lock file maintenance", "Awaiting Schedule", false},
			{"unpend", "renovate/toolchain", "chore(deps): update dependency maestro to cli-2.12.0", "Pending Status Checks", false},
			{"unpend", "renovate/github-actions", "chore(deps): update ghcr.io/renovatebot/renovate docker tag to v44.198.2", "Pending Status Checks", false},
		}}},
		{"renovate-config", Dashboard{Items: []Item{
			{"approve", "renovate/major-github-actions-major", "chore(deps): update dependency ubuntu to v26", "Pending Approval", false},
			{"unpend", "renovate/github-actions", "chore(deps): update ghcr.io/renovatebot/renovate docker tag to v44.198.2", "Pending Status Checks", false},
		}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.Open("testdata/" + tt.name + ".md")
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			got, err := Parse(f)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestParseCheckboxes(t *testing.T) {
	input := "## Awaiting Schedule\r\n" +
		" - [x] <!-- unschedule-branch=renovate/a --> **Update `a`** \r\n" +
		"- [X] <!-- unpend-branch=renovate/b -->Update b\r\n" +
		"- [ ] <!-- approve-branch=renovate/c --> Update c\r\n" +
		"- [x] <!-- approve-all-pending-prs --> All"
	want := Dashboard{Items: []Item{
		{"unschedule", "renovate/a", "**Update `a`**", "Awaiting Schedule", true},
		{"unpend", "renovate/b", "Update b", "Awaiting Schedule", true},
		{"approve", "renovate/c", "Update c", "Awaiting Schedule", false},
	}, BulkActions: []BulkAction{{"approve-all-pending-prs", true}}}
	got, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestParseIgnoresUnsupportedControls(t *testing.T) {
	for _, line := range []string{
		"- [ ] <!-- future-branch=renovate/a --> Update",
		"- [ ] <!-- prefix-approve-branch=renovate/a --> Update",
		"- [ ] <!-- approve-branch= --> Update",
		"- [ ] <!-- approve-branch=renovate/a extra --> Update",
		"- [ ] <!-- approve-all-pending-prs-extra --> All",
		"- [ ] <!-- future-all-pending-prs --> All",
		"- [ ] <!-- approve --> Update",
		"- [ ] <!-- approve-branch=renovate/a Update",
		"- [ ] Ordinary checkbox",
		"- [y] <!-- approve-branch=renovate/a --> Update",
		"<!-- approve-branch=renovate/a --> Update",
	} {
		t.Run(line, func(t *testing.T) {
			got, err := Parse(strings.NewReader("## Pending Approval\n" + line))
			if err != nil {
				t.Fatal(err)
			}
			if len(got.Items) != 0 || len(got.BulkActions) != 0 {
				t.Errorf("unexpected operations: %#v", got)
			}
		})
	}
}

func TestParseDetectedDependencies(t *testing.T) {
	input := `## Detected Dependencies
- [ ] <!-- approve-branch=renovate/ignored --> Ignored
### Nested heading
- [x] <!-- approve-all-pending-prs --> Ignored
## Pending Status Checks
- [ ] <!-- unpend-branch=renovate/a --> Update
`
	got, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	want := Dashboard{Items: []Item{{"unpend", "renovate/a", "Update", "Pending Status Checks", false}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestParseEmpty(t *testing.T) {
	got, err := Parse(strings.NewReader(""))
	if err != nil || !reflect.DeepEqual(got, Dashboard{}) {
		t.Fatalf("got %#v, %v", got, err)
	}
}

func TestParseLongLine(t *testing.T) {
	title := strings.Repeat("a", 128*1024)
	got, err := Parse(strings.NewReader("## Pending Approval\n- [ ] <!-- approve-branch=renovate/a -->" + title))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 || got.Items[0].Title != title {
		t.Fatal("long title was not preserved")
	}
}

var errRead = errors.New("read failed")

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errRead }

func TestParseReadError(t *testing.T) {
	input := strings.NewReader("## Pending Approval\n- [ ] <!-- approve-branch=renovate/a --> Update\n- [ ] <!-- retry-branch=renovate/b --> Retry\n")
	got, err := Parse(io.MultiReader(input, failingReader{}))
	if !errors.Is(err, errRead) {
		t.Fatalf("got %v, want read error", err)
	}
	if !reflect.DeepEqual(got, Dashboard{}) {
		t.Errorf("returned partial result: %#v", got)
	}
}

func TestParseUnknownControls(t *testing.T) {
	input := "## Pending Approval\n"
	controls := []string{"retry-branch=renovate/foo", "unlimit-branch=renovate/bar", "rebase-branch=renovate/baz", "approve-branch=", "approve-all-pending-prs-extra", "retry-branch=renovate/foo"}
	for _, control := range controls {
		input += "- [x] <!-- " + control + " --> Update\n"
	}
	input += "- [ ] <!-- approve-branch=renovate/known --> Known\n"
	got, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	want := Dashboard{
		Items:           []Item{{"approve", "renovate/known", "Known", "Pending Approval", false}},
		UnknownControls: controls,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestParseFencedCode(t *testing.T) {
	for _, tt := range []struct{ name, open, inside, close string }{
		{"backticks", "```markdown", "", "```"},
		{"tildes", "~~~markdown", "", "~~~"},
		{"long fence", "````markdown", "```\n", "`````"},
		{"different marker", "```", "~~~\n", "```"},
		{"closing info is not a close", "```", "```markdown\n", "```"},
		{"indented fence", "   ~~~markdown", "", "  ~~~"},
		{"unclosed fence", "```", "", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := "## Pending Approval\n" + tt.open + "\n" + tt.inside +
				"- [ ] <!-- approve-branch=renovate/example --> Example\n" +
				"- [x] <!-- approve-all-pending-prs --> All\n" +
				"- [ ] <!-- retry-branch=renovate/example --> Retry\n" +
				"## Update review policy\n" + tt.close + "\n" +
				"- [ ] <!-- approve-branch=renovate/real --> Real\n"
			got, err := Parse(strings.NewReader(input))
			if err != nil {
				t.Fatal(err)
			}
			want := Dashboard{}
			if tt.close != "" {
				want.Items = []Item{{"approve", "renovate/real", "Real", "Pending Approval", false}}
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("got %#v, want %#v", got, want)
			}
		})
	}
}

func TestParseIgnoresNonOperationSections(t *testing.T) {
	for _, heading := range []string{"", "## Update review policy", "## Detected Dependencies", "## Other Updates", "# Pending Approval"} {
		t.Run(heading, func(t *testing.T) {
			input := heading + "\n- [ ] <!-- approve-branch=renovate/example --> Example\n" +
				"- [ ] <!-- approve-all-pending-prs --> All\n" +
				"- [ ] <!-- retry-branch=renovate/example --> Retry\n"
			got, err := Parse(strings.NewReader(input))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, Dashboard{}) {
				t.Errorf("unexpected result: %#v", got)
			}
		})
	}
	// A policy heading must end the preceding operation section.
	got, err := Parse(strings.NewReader("## Pending Approval\n## Update review policy\n```markdown\n- [ ] <!-- approve-branch=renovate/example --> Example\n```\n- [ ] <!-- approve-branch=renovate/example --> Example"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, Dashboard{}) {
		t.Errorf("unexpected result: %#v", got)
	}
}

func TestParseAdditionalOperationSections(t *testing.T) {
	tests := []struct{ section, control string }{
		{"Rate-Limited", "unlimit-branch=renovate/example"},
		{"Errored", "retry-branch=renovate/example"},
		{"Open", "rebase-branch=renovate/example"},
		{"PR Edited (Blocked)", "rebase-branch=renovate/example"},
		{"PR Closed (Blocked)", "recreate-branch=renovate/example"},
		{"Group Size Not Met", "approveGroup-branch=renovate/example"},
		{"PR Creation Approval Required", "approvePr-branch=renovate/example"},
		{"Pending Branch Automerge", "approvePr-branch=renovate/example"},
		{"Other Branches", "other-branch=renovate/example"},
		{"Config Migration Needed", "create-config-migration-pr"},
		{"Config Migration Needed (Blocked)", "future-control"},
	}
	for _, tt := range tests {
		t.Run(tt.section, func(t *testing.T) {
			input := "## " + tt.section + "\n\n### Category\n\n" +
				" - [ ] <!-- " + tt.control + " --> Example\n" +
				" - [x] <!-- " + tt.control + " --> Example\n" +
				" - [ ] <!-- future-control --> Future\n" +
				"```markdown\n- [ ] <!-- retry-branch=renovate/code --> Code\n```\n" +
				"    - [ ] <!-- retry-branch=renovate/indented-code --> Code\n" +
				"## Update review policy\n- [ ] <!-- retry-branch=renovate/policy --> Policy\n"
			got, err := Parse(strings.NewReader(input))
			if err != nil {
				t.Fatal(err)
			}
			want := Dashboard{UnknownControls: []string{tt.control, tt.control, "future-control"}}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("got %#v, want %#v", got, want)
			}
		})
	}
}

func TestParseUnsupportedBulkActions(t *testing.T) {
	input := "## Rate-Limited\n- [ ] <!-- create-all-rate-limited-prs --> All\n" +
		"## Awaiting Schedule\n- [ ] <!-- create-all-awaiting-schedule-prs --> All\n" +
		"## Open\n- [ ] <!-- rebase-all-open-prs --> All\n"
	got, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	want := Dashboard{UnknownControls: []string{"create-all-rate-limited-prs", "create-all-awaiting-schedule-prs", "rebase-all-open-prs"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestParseCheckboxIndentation(t *testing.T) {
	for _, indent := range []string{"", " ", "  ", "   ", "    ", "     ", "\t", " \t"} {
		t.Run(indent, func(t *testing.T) {
			input := "## Pending Approval\n" +
				indent + "- [ ] <!-- approve-branch=renovate/example --> Example\n" +
				indent + "- [ ] <!-- retry-branch=renovate/example --> Retry\n" +
				indent + "- [ ] <!-- approve-all-pending-prs --> All\n"
			got, err := Parse(strings.NewReader(input))
			if err != nil {
				t.Fatal(err)
			}
			want := Dashboard{}
			if len(indent) <= 3 && !strings.Contains(indent, "\t") {
				want = Dashboard{
					Items:           []Item{{"approve", "renovate/example", "Example", "Pending Approval", false}},
					BulkActions:     []BulkAction{{"approve-all-pending-prs", false}},
					UnknownControls: []string{"retry-branch=renovate/example"},
				}
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("got %#v, want %#v", got, want)
			}
		})
	}
}
