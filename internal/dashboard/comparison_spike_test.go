package dashboard

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// This experiment compares internal records directly to expose comparison rules.
// Production tests should usually check the resulting Research instead.
func TestComparisonSpikeUnexportedFields(t *testing.T) {
	tests := []struct {
		name                string
		want, got           dependencyRecord
		equal, ignoredEqual bool
	}{
		{name: "equal", want: dependencyRecord{name: "node"}, got: dependencyRecord{name: "node"}, equal: true, ignoredEqual: true},
		{name: "private-name", want: dependencyRecord{name: "node"}, got: dependencyRecord{name: "pnpm"}, ignoredEqual: true},
		{name: "private-updates", want: dependencyRecord{updates: []string{"2"}}, got: dependencyRecord{updates: []string{"3"}}, ignoredEqual: true},
		{name: "private-nil-empty", got: dependencyRecord{updates: []string{}}, ignoredEqual: true},
		{name: "exported-version", want: dependencyRecord{DetectedDependency: DetectedDependency{CurrentVersion: "1"}}, got: dependencyRecord{DetectedDependency: DetectedDependency{CurrentVersion: "2"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := reflect.DeepEqual(tt.want, tt.got); got != tt.equal {
				t.Errorf("DeepEqual = %v, want %v", got, tt.equal)
			}
			// Even code in the same package needs a go-cmp option for these fields.
			func() {
				defer func() {
					if r := recover(); r == nil {
						t.Error("default cmp.Diff did not panic")
					} else {
						t.Logf("default cmp.Diff panic: %v", r)
					}
				}()
				cmp.Diff(tt.want, tt.got)
			}()
			diff := cmp.Diff(tt.want, tt.got, cmp.AllowUnexported(dependencyRecord{}))
			if got := diff == ""; got != tt.equal {
				t.Errorf("AllowUnexported equality = %v, want %v; diff:\n%s", got, tt.equal, diff)
			}
			ignored := cmp.Diff(tt.want, tt.got, cmpopts.IgnoreUnexported(dependencyRecord{}))
			if got := ignored == ""; got != tt.ignoredEqual {
				t.Errorf("IgnoreUnexported equality = %v, want %v", got, tt.ignoredEqual)
			}
			t.Logf("DeepEqual/AllowUnexported equality: %v; IgnoreUnexported equality: %v", tt.equal, tt.ignoredEqual)
		})
	}
}
