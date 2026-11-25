package command

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

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
	} {
		t.Run(tt.name, func(t *testing.T) {
			var out, diagnostic bytes.Buffer
			if code := Run(tt.args, tt.input, &out, &diagnostic); code == 0 || out.Len() != 0 || !strings.Contains(diagnostic.String(), tt.message) {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), diagnostic.String())
			}
		})
	}
}

func TestHelp(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"-h"}} {
		var out, diagnostic bytes.Buffer
		if code := Run(args, failingIO{}, &out, &diagnostic); code != 0 || !strings.Contains(out.String(), "Usage: renodash parse") || diagnostic.Len() != 0 {
			t.Fatalf("args=%v code=%d stdout=%q stderr=%q", args, code, out.String(), diagnostic.String())
		}
	}
}
