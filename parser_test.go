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
			{"approve", "renovate/toolchain", "chore(deps): update dependency pnpm to v12.14.0", "Pending Approval", false, nil, ""},
			{"approve", "renovate/major-github-actions-major", "chore(deps): update dependency ubuntu to v26", "Pending Approval", false, nil, ""},
			{"approve", "renovate/lock-file-maintenance-toolchain", "chore(deps): lock file maintenance", "Pending Approval", false, nil, ""},
			{"unpend", "renovate/major-toolchain", "chore(deps): update node.js to v26", "Pending Status Checks", false, nil, ""},
		}, BulkActions: []BulkAction{{"approve-all-pending-prs", false}}}},
		{"maestro-playground", Dashboard{Items: []Item{
			{"approve", "renovate/major-github-actions-major", "chore(deps): update dependency ubuntu to v26", "Pending Approval", false, nil, ""},
			{"unschedule", "renovate/lock-file-maintenance-toolchain", "chore(deps): lock file maintenance", "Awaiting Schedule", false, nil, ""},
			{"unpend", "renovate/toolchain", "chore(deps): update dependency maestro to cli-2.12.0", "Pending Status Checks", false, nil, ""},
			{"unpend", "renovate/github-actions", "chore(deps): update ghcr.io/renovatebot/renovate docker tag to v44.198.2", "Pending Status Checks", false, nil, ""},
		}}},
		{"renovate-config", Dashboard{Items: []Item{
			{"approve", "renovate/major-github-actions-major", "chore(deps): update dependency ubuntu to v26", "Pending Approval", false, nil, ""},
			{"unpend", "renovate/github-actions", "chore(deps): update ghcr.io/renovatebot/renovate docker tag to v44.198.2", "Pending Status Checks", false, nil, ""},
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
			for i := range got.Items {
				got.Items[i].Research = nil
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
		{"unschedule", "renovate/a", "**Update `a`**", "Awaiting Schedule", true, nil, ""},
		{"unpend", "renovate/b", "Update b", "Awaiting Schedule", true, nil, ""},
		{"approve", "renovate/c", "Update c", "Awaiting Schedule", false, nil, ""},
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
	want := Dashboard{Items: []Item{{"unpend", "renovate/a", "Update", "Pending Status Checks", false, nil, ""}}}
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
		Items:           []Item{{"approve", "renovate/known", "Known", "Pending Approval", false, nil, ""}},
		UnknownControls: controls,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestParseOpenPRControls(t *testing.T) {
	input := `## Open
 - [ ] <!-- rebase-branch=renovate/mise-cli -->[chore(deps): update mise cli to v2026.10.15](../pull/32) (` + "`mise`" + `)
 - [x] <!-- rebase-branch=renovate/flutter -->[chore(deps): update dependency flutter to v3.47.7](../pull/33)
 - [ ] <!-- rebase-all-open-prs -->**Rebase all**
 - [ ] <!-- retry-branch=renovate/other -->Other
`
	got, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	for i := range got.Items {
		got.Items[i].Research = nil
	}
	want := Dashboard{
		Items: []Item{
			{Action: "rebase", Branch: "renovate/mise-cli", Title: "[chore(deps): update mise cli to v2026.10.15](../pull/32) (`mise`)", PRLink: "../pull/32", Section: "Open"},
			{Action: "rebase", Branch: "renovate/flutter", Title: "[chore(deps): update dependency flutter to v3.47.7](../pull/33)", PRLink: "../pull/33", Section: "Open", Checked: true},
		},
		BulkActions:     []BulkAction{{Action: "rebase-all-open-prs"}},
		UnknownControls: []string{"retry-branch=renovate/other"},
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
				want.Items = []Item{{"approve", "renovate/real", "Real", "Pending Approval", false, nil, ""}}
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
		{"Open", "retry-branch=renovate/example"},
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
	want := Dashboard{BulkActions: []BulkAction{{Action: "rebase-all-open-prs"}}, UnknownControls: []string{"create-all-rate-limited-prs", "create-all-awaiting-schedule-prs"}}
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
					Items:           []Item{{"approve", "renovate/example", "Example", "Pending Approval", false, nil, ""}},
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

func TestParseHTMLCommentBlocks(t *testing.T) {
	tests := []struct {
		name, body string
		want       Dashboard
	}{
		{"hidden checkbox", "<!--\n- [ ] <!-- approve-branch=renovate/hidden --> Hidden\n-->\n", Dashboard{}},
		{"hidden bulk action", "<!--\n- [ ] <!-- approve-all-pending-prs --> Hidden\n-->\n", Dashboard{}},
		{"hidden unknown control", "<!--\n- [ ] <!-- retry-branch=renovate/hidden --> Hidden\n-->\n", Dashboard{}},
		{"hidden headings and fences", " <!-- explanation\n## Update review policy\n```markdown\n~~~\n-->\n", Dashboard{}},
		{"single line", "<!-- explanation -->\n", Dashboard{}},
		{"unclosed", "<!-- explanation\n", Dashboard{}},
		{"first delimiter closes", "<!--\n- [ ] <!-- retry-branch=renovate/hidden --> Hidden\n- [ ] <!-- retry-branch=renovate/visible --> Visible\n", Dashboard{UnknownControls: []string{"retry-branch=renovate/visible"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := "## Pending Approval\n" + tt.body + "- [ ] <!-- approve-branch=renovate/real --> Real\n"
			want := tt.want
			if tt.name != "unclosed" {
				want.Items = []Item{{"approve", "renovate/real", "Real", "Pending Approval", false, nil, ""}}
			}
			got, err := Parse(strings.NewReader(input))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("got %#v, want %#v", got, want)
			}
		})
	}
}

func TestParseHTMLCommentInsideFence(t *testing.T) {
	input := "## Pending Approval\n```markdown\n<!-- unclosed comment example\n```\n- [ ] <!-- approve-branch=renovate/real --> Real\n"
	got, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	want := Dashboard{Items: []Item{{"approve", "renovate/real", "Real", "Pending Approval", false, nil, ""}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestParseClosingHeadingHashes(t *testing.T) {
	for _, heading := range []string{"## Pending Approval ##", "## Pending Approval #", "## Pending Approval ###   ", "  ## Pending Approval\t##\t"} {
		t.Run(heading, func(t *testing.T) {
			input := heading + "\n- [ ] <!-- approve-branch=renovate/real --> Real\n" +
				"## Rate-Limited ##\n- [ ] <!-- unlimit-branch=renovate/a --> Update\n" +
				"## Detected Dependencies ##\n- [ ] <!-- approve-branch=renovate/hidden --> Hidden\n"
			got, err := Parse(strings.NewReader(input))
			if err != nil {
				t.Fatal(err)
			}
			want := Dashboard{
				Items:           []Item{{"approve", "renovate/real", "Real", "Pending Approval", false, nil, ""}},
				UnknownControls: []string{"unlimit-branch=renovate/a"},
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("got %#v, want %#v", got, want)
			}
		})
	}
	// Without whitespace before the hashes, they are part of the heading text.
	got, err := Parse(strings.NewReader("## Pending Approval##\n- [ ] <!-- approve-branch=renovate/hidden --> Hidden"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, Dashboard{}) {
		t.Errorf("unexpected result: %#v", got)
	}
}
