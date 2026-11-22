package testingcomparison

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/gkzz/renodash/internal/dashboard"
)

// Each implementation returns exactly the message its assertion would report.
func standardMessage(want, got dashboard.Dashboard) string {
	if reflect.DeepEqual(want, got) {
		return ""
	}
	return fmt.Sprintf("mismatch:\nwant: %#v\n got: %#v", want, got)
}

// Keep the same whole-value equality check; improve only its failure message.
func standardDetailedMessage(want, got dashboard.Dashboard) string {
	message := standardMessage(want, got)
	if message == "" {
		return ""
	}
	var details strings.Builder
	details.WriteString(message)
	for i := 0; i < min(len(want.Items), len(got.Items)); i++ {
		w, g := want.Items[i].Research, got.Items[i].Research
		if reflect.DeepEqual(w, g) {
			continue
		}
		// Print each pointer as a top-level value: fmt then shows its contents.
		fmt.Fprintf(&details, "\nItems[%d].Research:\nwant: %#v\n got: %#v", i, w, g)
	}
	return details.String()
}

func TestStandardDetailedDiagnostics(t *testing.T) {
	tests := []struct {
		name      string
		want, got dashboard.Dashboard
		contains  []string
	}{
		{
			name:     "nested-package",
			want:     dashboard.Dashboard{Items: []dashboard.Item{{Research: &dashboard.Research{Package: "want-package"}}}},
			got:      dashboard.Dashboard{Items: []dashboard.Item{{Research: &dashboard.Research{Package: "got-package"}}}},
			contains: []string{"Items[0].Research", "Package:", "want-package", "got-package"},
		},
		{
			name:     "grouped-evidence",
			want:     dashboard.Dashboard{Items: []dashboard.Item{{Research: &dashboard.Research{Packages: []dashboard.GroupPackageResearch{{DetectedDependencies: []dashboard.GroupDetectedDependency{{DetectedDependency: dashboard.DetectedDependency{CurrentVersion: "want-version"}}}}}}}}},
			got:      dashboard.Dashboard{Items: []dashboard.Item{{Research: &dashboard.Research{Packages: []dashboard.GroupPackageResearch{{DetectedDependencies: []dashboard.GroupDetectedDependency{{DetectedDependency: dashboard.DetectedDependency{CurrentVersion: "got-version"}}}}}}}}},
			contains: []string{"Items[0].Research", "CurrentVersion:", "want-version", "got-version"},
		},
		{
			name:     "nested-nil-empty",
			want:     dashboard.Dashboard{Items: []dashboard.Item{{Research: &dashboard.Research{}}}},
			got:      dashboard.Dashboard{Items: []dashboard.Item{{Research: &dashboard.Research{ReferenceURLs: []string{}}}}},
			contains: []string{"Items[0].Research", "[]string(nil)", "[]string{}"},
		},
		{
			name:     "nil-research",
			want:     dashboard.Dashboard{Items: []dashboard.Item{{}}},
			got:      dashboard.Dashboard{Items: []dashboard.Item{{Research: &dashboard.Research{}}}},
			contains: []string{"Items[0].Research", "nil"},
		},
		{name: "nil-items", got: dashboard.Dashboard{Items: []dashboard.Item{}}, contains: []string{"Items:", "nil"}},
		{name: "extra-item", got: dashboard.Dashboard{Items: []dashboard.Item{{Title: "extra"}}}, contains: []string{"extra"}},
		{name: "item-title", want: dashboard.Dashboard{Items: []dashboard.Item{{Title: "want-title"}}}, got: dashboard.Dashboard{Items: []dashboard.Item{{Title: "got-title"}}}, contains: []string{"want-title", "got-title"}},
		{name: "bulk-action", got: dashboard.Dashboard{BulkActions: []dashboard.BulkAction{{Action: "approve"}}}, contains: []string{"BulkActions:", "approve"}},
		{name: "unknown-control", got: dashboard.Dashboard{UnknownControls: []string{"unsupported"}}, contains: []string{"UnknownControls:", "unsupported"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := standardDetailedMessage(tt.want, tt.got)
			for _, text := range tt.contains {
				if !strings.Contains(message, text) {
					t.Errorf("diagnostic missing %q:\n%s", text, message)
				}
			}
			t.Log(message)
		})
	}
	for _, value := range []dashboard.Dashboard{{}, {Items: []dashboard.Item{{Research: &dashboard.Research{Package: "same"}}}}} {
		if message := standardDetailedMessage(value, value); message != "" {
			t.Errorf("equal values reported a mismatch: %s", message)
		}
	}
}

// Exercise the standard-only assertion against actual parsing, not only fixtures.
func TestStandardParserResearch(t *testing.T) {
	input := "## Pending Approval\n- [ ] <!-- approve-branch=a --> Update pnpm to v2\n" +
		"## Detected Dependencies\n<details><summary>mise</summary>\n" +
		"<details><summary>mise.toml</summary>\n- `pnpm 1`\n</details>\n</details>\n"
	want := dashboard.Dashboard{Items: []dashboard.Item{{
		Action: "approve", Branch: "a", Title: "Update pnpm to v2", Section: "Pending Approval",
		Research: &dashboard.Research{
			Package: "pnpm", TargetVersion: "v2",
			DetectedDependencies: []dashboard.DetectedDependency{{CurrentVersion: "1", Manager: "mise", File: "mise.toml"}},
			ReferenceURLs:        []string{},
			Provenance: dashboard.ResearchProvenance{
				PackageSource: "item_title", TargetVersionSource: "item_title",
				DetectedDependenciesSource: "detected_dependencies", Association: "exact_package_name", ReferenceURLSource: "item_title",
			},
		},
	}}}
	got, err := dashboard.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if message := standardDetailedMessage(want, got); message != "" {
		t.Error(message)
	}
}
