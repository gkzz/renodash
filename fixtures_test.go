package main

import (
	"os"
	"strings"
	"testing"
)

// These checks protect the fixture coverage until parser tests are added.
func TestDashboardFixtures(t *testing.T) {
	tests := []struct {
		name     string
		controls map[string]int
	}{
		{"dotfiles", map[string]int{"approve-branch=": 3, "approve-all-pending-prs": 1, "unpend-branch=": 1}},
		{"maestro-playground", map[string]int{"approve-branch=": 1, "unschedule-branch=": 1, "unpend-branch=": 2}},
		{"renovate-config", map[string]int{"approve-branch=": 1, "unpend-branch=": 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := os.ReadFile("testdata/" + tt.name + ".md")
			if err != nil {
				t.Fatal(err)
			}
			for control, want := range tt.controls {
				if got := strings.Count(string(body), " - [ ] <!-- "+control); got != want {
					t.Errorf("control %q: got %d checkboxes, want %d", control, got, want)
				}
			}
		})
	}
}
