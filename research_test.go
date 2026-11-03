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
	want := &Research{Package: "pnpm", TargetVersion: "v12.14.0", DetectedDependencies: []DetectedDependency{{"12.13.0", "mise", "mise.toml"}}, ReferenceURLs: []string{}, Provenance: provenance(false)}
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
	want := &Research{Package: "pnpm", TargetVersion: "v2", DetectedDependencies: []DetectedDependency{{"1", "mise", "mise.toml"}, {"0.9", "mise", "other.toml"}}, ReferenceURLs: []string{"https://example.com/pnpm"}, Provenance: provenance(false)}
	if !reflect.DeepEqual(d.Items[0].Research, want) {
		t.Fatalf("got %#v, want %#v", d.Items[0].Research, want)
	}
	if d.Items[1].Research != nil {
		t.Fatal("group update must remain unresolved")
	}
}

func TestGroupedResearchDotfiles(t *testing.T) {
	f, err := os.Open("testdata/grouped-dotfiles.md")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	d, err := Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	r := d.Items[0].Research
	if r == nil {
		t.Fatal("missing group research")
	}
	if r.Provenance != provenance(true) {
		t.Fatalf("group provenance: %#v", r.Provenance)
	}
	names := []string{}
	for _, p := range r.Packages {
		names = append(names, p.Package)
		if p.TargetVersion != "unknown" {
			t.Fatalf("inferred target: %#v", p)
		}
	}
	if !reflect.DeepEqual(names, []string{"go", "node", "pnpm", "python"}) {
		t.Fatalf("members: %v", names)
	}
	want := []GroupDetectedDependency{{DetectedDependency{"24.22.1", "mise", "mise.toml"}, []string{"24.22.2", "26.13.0"}}}
	if !reflect.DeepEqual(r.Packages[1].DetectedDependencies, want) {
		t.Fatalf("node evidence: %#v", r.Packages[1])
	}
	if got := r.Packages[0].DetectedDependencies[0].AvailableUpdates; !reflect.DeepEqual(got, []string{"1.27.3"}) {
		t.Fatalf("go updates: %v", got)
	}
}

func TestGroupedResearchEvidence(t *testing.T) {
	input := "## Pending Approval\n- [ ] <!-- approve-branch=a --> Update tools (`node`, `node.js`, `missing`, `node`)\n" +
		"## Detected Dependencies\n<details><summary>mise</summary>\n<details><summary>mise.toml</summary>\n" +
		"- `node 1` → [Updates: `2`, `3`]\n- `node 1` → [Updates: `2`, `3`]\n- `node 1` → [Updates: `2`]\n</details>\n</details>\n" +
		"<details><summary>npm</summary>\n<details><summary>package.json</summary>\n- `node 0.9`\n</details>\n</details>\n" +
		"```\n<details><summary>fake</summary>\n<details><summary>fake</summary>\n- `node fake`\n```\n"
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	packages := d.Items[0].Research.Packages
	if len(packages) != 3 {
		t.Fatalf("packages: %#v", packages)
	}
	want := []GroupDetectedDependency{
		{DetectedDependency{"1", "mise", "mise.toml"}, []string{"2", "3"}},
		{DetectedDependency{"1", "mise", "mise.toml"}, []string{"2"}},
		{DetectedDependency{"0.9", "npm", "package.json"}, []string{}},
	}
	if !reflect.DeepEqual(packages[0].DetectedDependencies, want) {
		t.Fatalf("evidence: %#v", packages[0])
	}
	for _, p := range packages[1:] {
		if len(p.DetectedDependencies) != 0 {
			t.Fatalf("unexpected association: %#v", p)
		}
	}
}

func TestGroupedResearchUnsupportedTitles(t *testing.T) {
	for _, title := range []string{"Update tools", "Update tools (node, python)", "Update tools (`node`, python)", "Lock file maintenance (`node`)", "Update tools (`node`,)", "Update tools (`node` to `2`)", "Update dependency foo to v2 (`bar`)", "Update foo to v2 (`bar`)", "chore(deps): update `foo` docker tag to `v2` (`bar`)", "Update dependency [foo](https://example.com/foo) to v2 (`bar`)"} {
		t.Run(title, func(t *testing.T) {
			d, err := Parse(strings.NewReader("## Pending Approval\n- [ ] <!-- approve-branch=a --> " + title))
			if err != nil {
				t.Fatal(err)
			}
			if d.Items[0].Research != nil {
				t.Fatalf("unexpected research: %#v", d.Items[0].Research)
			}
		})
	}
}

func TestGroupedResearchSingleMemberTrailingWhitespace(t *testing.T) {
	for _, suffix := range []string{"", "  ", "\t", " \t "} {
		t.Run("suffix="+suffix, func(t *testing.T) {
			input := "## Pending Approval\n- [ ] <!-- approve-branch=a --> Update tools (`node`)\n" +
				"## Detected Dependencies\n<details><summary>mise</summary>\n<details><summary>mise.toml</summary>\n" +
				"- `node 24.22.1` → [Updates: `24.22.2`, `26.13.0`]" + suffix + "\n</details>\n</details>\n"
			d, err := Parse(strings.NewReader(input))
			if err != nil {
				t.Fatal(err)
			}
			r := d.Items[0].Research
			want := []GroupPackageResearch{{Package: "node", TargetVersion: "unknown", DetectedDependencies: []GroupDetectedDependency{{DetectedDependency{"24.22.1", "mise", "mise.toml"}, []string{"24.22.2", "26.13.0"}}}}}
			if r == nil || !reflect.DeepEqual(r.Packages, want) {
				t.Fatalf("research: %#v; want packages %#v", r, want)
			}
		})
	}
}
