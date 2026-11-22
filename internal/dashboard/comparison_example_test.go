package dashboard

import (
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Run both comparisons with:
// go test ./internal/dashboard -run '^TestParseComparison' -v
// Both tests use testing; only the equality check and failure message differ.
func parseComparisonFixture(t *testing.T) (Dashboard, Dashboard) {
	t.Helper()
	input := "## Pending Approval\n" +
		"- [ ] <!-- approve-branch=a --> Update pnpm to v12.14.0\n" +
		"## Detected Dependencies\n" +
		"<details><summary>mise</summary>\n" +
		"<details><summary>mise.toml</summary>\n" +
		"- `pnpm 12.13.0`\n</details>\n</details>\n"
	got, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	want := Dashboard{Items: []Item{{
		Action: "approve", Branch: "a",
		Title: "Update pnpm to v12.14.0", Section: "Pending Approval",
		Research: &Research{
			Package: "pnpm", TargetVersion: "v12.14.0",
			DetectedDependencies: []DetectedDependency{{
				CurrentVersion: "12.13.0", Manager: "mise", File: "mise.toml",
			}},
			ReferenceURLs: []string{},
			Provenance: ResearchProvenance{
				PackageSource: "item_title", TargetVersionSource: "item_title",
				DetectedDependenciesSource: "detected_dependencies",
				Association:                "exact_package_name", ReferenceURLSource: "item_title",
			},
		},
	}}}
	// Uncomment to compare failure messages for the same nested mismatch.
	// The standard message prints Research as an address; cmp shows CurrentVersion.
	// got.Items[0].Research.DetectedDependencies[0].CurrentVersion = "12.12.0"
	return want, got
}

func TestParseComparisonStandard(t *testing.T) {
	want, got := parseComparisonFixture(t)
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("dashboard mismatch\nwant: %#v\ngot:  %#v", want, got)
	}
}

func TestParseComparisonCmp(t *testing.T) {
	want, got := parseComparisonFixture(t)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("dashboard mismatch (-want +got):\n%s", diff)
	}
}
