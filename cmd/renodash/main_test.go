package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

func TestFixtures(t *testing.T) {
	for _, name := range []string{"dotfiles", "maestro-playground", "renovate-config", "grouped-dotfiles"} {
		for _, args := range [][]string{{"parse"}, {"parse", "--format", "json"}} {
			t.Run(name+strings.Join(args, "_"), func(t *testing.T) {
				input, err := os.Open("../../testdata/" + name + ".md")
				if err != nil {
					t.Fatal(err)
				}
				defer input.Close()
				want, err := os.ReadFile("testdata/" + name + ".json")
				if err != nil {
					t.Fatal(err)
				}
				var out, diagnostic bytes.Buffer
				code := run(args, input, &out, &diagnostic)
				if code != 0 || diagnostic.Len() != 0 || out.String() != string(want) {
					t.Fatalf("code=%d stderr=%q\ngot:\n%s\nwant:\n%s", code, diagnostic.String(), out.String(), want)
				}
			})
		}
	}
}

func TestEmptyDashboard(t *testing.T) {
	var out, diagnostic bytes.Buffer
	code := run([]string{"parse"}, strings.NewReader(""), &out, &diagnostic)
	const want = "{\n  \"items\": [],\n  \"bulk_actions\": [],\n  \"unknown_controls\": []\n}\n"
	if code != 0 || diagnostic.Len() != 0 || out.String() != want {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), diagnostic.String())
	}
}

func TestUnknownControls(t *testing.T) {
	input := "## Pending Approval\n- [x] <!-- approve-branch=renovate/a --> Update a\n- [ ] <!-- retry-branch=renovate/example --> Retry\n- [ ] <!-- future --> Future\n"
	var out, diagnostic bytes.Buffer
	code := run([]string{"parse"}, strings.NewReader(input), &out, &diagnostic)
	const want = `{
  "items": [
    {
      "action": "approve",
      "branch": "renovate/a",
      "title": "Update a",
      "section": "Pending Approval",
      "checked": true
    }
  ],
  "bulk_actions": [],
  "unknown_controls": [
    "retry-branch=renovate/example",
    "future"
  ]
}
`
	const warnings = "warning: unsupported Renovate control: retry-branch=renovate/example\nwarning: unsupported Renovate control: future\n"
	if code != 0 || out.String() != want || diagnostic.String() != warnings {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), diagnostic.String())
	}
}

type failingIO struct{}

func (failingIO) Read([]byte) (int, error)  { return 0, errors.New("read failed") }
func (failingIO) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestErrors(t *testing.T) {
	for _, tt := range []struct {
		name    string
		args    []string
		input   io.Reader
		message string
	}{
		{"no command", nil, strings.NewReader(""), "expected subcommand"},
		{"unknown command", []string{"unknown"}, strings.NewReader(""), "unknown subcommand"},
		{"unsupported format", []string{"parse", "--format", "yaml"}, strings.NewReader(""), "unsupported output format"},
		{"missing format", []string{"parse", "--format"}, strings.NewReader(""), "flag needs an argument"},
		{"unknown flag", []string{"parse", "--unknown"}, strings.NewReader(""), "flag provided but not defined"},
		{"extra argument", []string{"parse", "extra"}, strings.NewReader(""), "unexpected argument"},
		{"read error", []string{"parse"}, failingIO{}, "read failed"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var out, diagnostic bytes.Buffer
			if code := run(tt.args, tt.input, &out, &diagnostic); code == 0 || out.Len() != 0 || !strings.Contains(diagnostic.String(), tt.message) {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), diagnostic.String())
			}
		})
	}
}

func TestWriteError(t *testing.T) {
	var diagnostic bytes.Buffer
	if code := run([]string{"parse"}, strings.NewReader(""), failingIO{}, &diagnostic); code == 0 || !strings.Contains(diagnostic.String(), "write JSON") {
		t.Fatalf("code=%d stderr=%q", code, diagnostic.String())
	}
}

func TestHelp(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"parse", "--help"}, {"-h"}, {"parse", "-h"}} {
		var out, diagnostic bytes.Buffer
		if code := run(args, failingIO{}, &out, &diagnostic); code != 0 || !strings.Contains(out.String(), "Usage: renodash parse") || diagnostic.Len() != 0 {
			t.Fatalf("args=%v code=%d stdout=%q stderr=%q", args, code, out.String(), diagnostic.String())
		}
	}
}
