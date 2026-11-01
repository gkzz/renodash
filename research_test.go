package renodash

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestResearchPnpmBeforePRCreation(t *testing.T) {
	f, err := os.Open("testdata/dotfiles.md")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	d, err := Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	want := &Research{Package: "pnpm", TargetVersion: "v12.14.0", DetectedDependencies: []DetectedDependency{{"12.13.0", "mise", "mise.toml"}}, ReferenceURLs: []string{}}
	if !reflect.DeepEqual(d.Items[0].Research, want) {
		t.Fatalf("got %#v, want %#v", d.Items[0].Research, want)
	}
	if d.Items[2].Research != nil {
		t.Fatal("lock maintenance must not be interpreted as a package update")
	}
	// node.js is a title alias: do not silently associate it with node.
	if len(d.Items[3].Research.DetectedDependencies) != 0 {
		t.Fatal("unexpected alias association")
	}
}

func TestResearchEvidence(t *testing.T) {
	input := "## Pending Approval\n" +
		"- [ ] <!-- approve-branch=a --> Update dependency [pnpm](https://example.com/pnpm) to `v2`\n" +
		"- [ ] <!-- approve-branch=b --> Update dependency group\n" +
		"## Detected Dependencies\n<details><summary>mise (2)</summary>\n<blockquote>\n" +
		"<details><summary>mise.toml (2)</summary>\n- `pnpm 1`\n- `pnpm 1`\n</details>\n" +
		"<details><summary>other.toml (1)</summary>\n- `pnpm 0.9`\n</details>\n</blockquote>\n</details>\n" +
		"```\n<details><summary>fake (1)</summary>\n<details><summary>fake.toml</summary>\n- `pnpm fake`\n```\n" +
		"## Policy\n- `pnpm ignored`\n"
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	want := &Research{Package: "pnpm", TargetVersion: "v2", DetectedDependencies: []DetectedDependency{{"1", "mise", "mise.toml"}, {"0.9", "mise", "other.toml"}}, ReferenceURLs: []string{"https://example.com/pnpm"}}
	if !reflect.DeepEqual(d.Items[0].Research, want) {
		t.Fatalf("got %#v, want %#v", d.Items[0].Research, want)
	}
	if d.Items[1].Research != nil {
		t.Fatal("group update must remain unresolved")
	}
}
