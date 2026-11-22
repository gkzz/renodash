// Package testingcomparison contains the spike's executable comparison only.
package testingcomparison

import (
	"strings"
	"testing"

	"github.com/gkzz/renodash/internal/dashboard"
	"github.com/google/go-cmp/cmp"
)

func cmpMessage(want, got dashboard.Dashboard) string {
	if diff := cmp.Diff(want, got); diff != "" {
		return "mismatch (-want +got):\n" + diff
	}
	return ""
}

func TestParserComparison(t *testing.T) {
	strategies := []struct {
		name    string
		message func(dashboard.Dashboard, dashboard.Dashboard) string
	}{
		{"standard", standardMessage},
		{"standard-detailed", standardDetailedMessage},
		{"go-cmp", cmpMessage},
	}
	for _, strategy := range strategies {
		t.Run(strategy.name, func(t *testing.T) {
			// Input and expectations are independently written, not generated
			// from the parser or from one another.
			tests := []struct {
				name  string
				input string
				want  dashboard.Dashboard
			}{
				{name: "empty"},
				{
					name:  "ordered-controls",
					input: "## Pending Approval\n- [ ] <!-- approve-branch=renovate/a --> Update a\n- [x] <!-- approve-branch=renovate/b --> Update b\n",
					want: dashboard.Dashboard{Items: []dashboard.Item{
						{Action: "approve", Branch: "renovate/a", Title: "Update a", Section: "Pending Approval"},
						{Action: "approve", Branch: "renovate/b", Title: "Update b", Section: "Pending Approval", Checked: true},
					}},
				},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					got, err := dashboard.Parse(strings.NewReader(tt.input))
					if err != nil {
						t.Fatal(err)
					}
					if message := strategy.message(tt.want, got); message != "" {
						t.Error(message)
					}
				})
			}
		})
	}
}

func TestMismatchDiagnostics(t *testing.T) {
	tests := []struct {
		name      string
		want, got dashboard.Dashboard
	}{
		{
			name: "nested-field",
			want: dashboard.Dashboard{Items: []dashboard.Item{{Research: &dashboard.Research{Package: "want-package"}}}},
			got:  dashboard.Dashboard{Items: []dashboard.Item{{Research: &dashboard.Research{Package: "got-package"}}}},
		},
		{name: "nil-versus-empty", want: dashboard.Dashboard{}, got: dashboard.Dashboard{Items: []dashboard.Item{}}},
		{
			name: "slice-order",
			want: dashboard.Dashboard{Items: []dashboard.Item{{Branch: "renovate/a"}, {Branch: "renovate/b"}}},
			got:  dashboard.Dashboard{Items: []dashboard.Item{{Branch: "renovate/b"}, {Branch: "renovate/a"}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, strategy := range []struct {
				name    string
				message func(dashboard.Dashboard, dashboard.Dashboard) string
			}{
				{"standard", standardMessage},
				{"standard-detailed", standardDetailedMessage},
				{"go-cmp", cmpMessage},
			} {
				t.Run(strategy.name, func(t *testing.T) {
					message := strategy.message(tt.want, tt.got)
					if message == "" {
						t.Fatal("comparison missed the expected mismatch")
					}
					if tt.name == "nested-field" && strategy.name != "standard" {
						// Check diagnostic content without depending on diff layout.
						for _, text := range []string{"Package", "want-package", "got-package"} {
							if !strings.Contains(message, text) {
								t.Errorf("diagnostic missing %q:\n%s", text, message)
							}
						}
					}
					t.Log(message)
				})
			}
		})
	}
}
