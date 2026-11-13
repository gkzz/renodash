package summary

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func TestGolden(t *testing.T) {
	// Captured from the original jq renderer before its removal. Never regenerate
	// these expectations from RenderOffline: they preserve the migration contract.
	for _, name := range []string{"empty", "escaping", "offline"} {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile("testdata/" + name + ".json")
			if err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile("testdata/" + name + ".golden")
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			if err := RenderOffline(bytes.NewReader(data), &out); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(out.Bytes(), want) {
				t.Fatalf("golden mismatch:\ngot:\n%s\nwant:\n%s", &out, want)
			}
		})
	}
}
func input(t *testing.T) map[string]any {
	t.Helper()
	data, err := os.ReadFile("testdata/empty.json")
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	return result
}
func render(t *testing.T, value map[string]any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := RenderOffline(bytes.NewReader(data), &out); err != nil {
		t.Fatal(err)
	}
	return out.String()
}
func TestCandidateLimits(t *testing.T) {
	for _, n := range []int{0, 1, 20, 21} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			value := input(t)
			items := []any{}
			for i := range n {
				items = append(items, map[string]any{"section": "Updates", "action": "update", "title": fmt.Sprintf("candidate-%02d", i)})
			}
			value["items"] = items
			value["bulk_actions"] = []any{map[string]any{}, map[string]any{}}
			value["unknown_controls"] = []any{map[string]any{}}
			out := render(t, value)
			if !strings.Contains(out, fmt.Sprintf("**Candidates:** %d", n)) || !strings.Contains(out, "**Bulk actions:** 2") || !strings.Contains(out, "**Unknown controls:** 1") {
				t.Fatal(out)
			}
			if strings.Contains(out, "No candidates") != (n == 0) || strings.Contains(out, "Showing the first 20") != (n > 20) {
				t.Fatal(out)
			}
			previous := -1
			for i := range min(n, 20) {
				position := strings.Index(out, fmt.Sprintf("candidate-%02d", i))
				if position <= previous {
					t.Fatal("candidate order changed", out)
				}
				previous = position
			}
			if strings.Contains(out, "candidate-20") {
				t.Fatal("candidate limit exceeded", out)
			}
		})
	}
}
func TestCheckFallbacks(t *testing.T) {
	value := input(t)
	value["pull_requests"] = []any{map[string]any{"number": 7, "url": "https://github.com/gkzz/renodash/pull/7", "checks": []any{
		map[string]any{"name": "preferred", "context": "ignored", "conclusion": "SUCCESS", "status": "COMPLETED", "state": "FAILURE"},
		map[string]any{"context": "status", "conclusion": "", "status": "PENDING", "state": "ERROR"},
		map[string]any{"name": nil, "context": "context", "conclusion": nil, "status": nil, "state": "FAILURE"},
		map[string]any{}, map[string]any{"name": "", "context": "ignored"},
	}}}
	out := render(t, value)
	want := "preferred: SUCCESS<br>status: PENDING<br>context: FAILURE<br>Unnamed check: Unknown<br>: Unknown"
	if !strings.Contains(out, want) {
		t.Fatal(out)
	}
}
func TestInvalidInput(t *testing.T) {
	for _, data := range []string{"", "{", "null", "[]", "{}", `{"source":null}`, `{"items":"wrong"}`} {
		t.Run(data, func(t *testing.T) {
			var out bytes.Buffer
			if err := RenderOffline(strings.NewReader(data), &out); err == nil || out.Len() != 0 {
				t.Fatalf("err=%v out=%s", err, &out)
			}
		})
	}
	valid, err := os.ReadFile("testdata/empty.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"{}", "null", "garbage", "{"} {
		var out bytes.Buffer
		if err := RenderOffline(strings.NewReader(string(valid)+suffix), &out); err == nil || out.Len() != 0 {
			t.Fatalf("suffix %q: err=%v out=%s", suffix, err, &out)
		}
	}
	var out bytes.Buffer
	if err := RenderOffline(strings.NewReader(string(valid)+" \n\t"), &out); err != nil {
		t.Fatal(err)
	}
}
func TestURLs(t *testing.T) {
	for _, raw := range []string{"", "javascript:alert(1)", "data:text/html,test", "/relative", "https:///missing", "https://github.com/\nx", "https://github.com/\rx", "https://github.com/\tx", "https://github.com/a b", "https://user@github.com/a", "https://github.com/%zz"} {
		for _, field := range []string{"issue", "pr"} {
			t.Run(field+raw, func(t *testing.T) {
				value := input(t)
				if field == "issue" {
					value["source"].(map[string]any)["issue_url"] = raw
				} else {
					value["pull_requests"] = []any{map[string]any{"number": 7, "url": raw}}
				}
				data, err := json.Marshal(value)
				if err != nil {
					t.Fatal(err)
				}
				var out bytes.Buffer
				if err := RenderOffline(bytes.NewReader(data), &out); err == nil || out.Len() != 0 {
					t.Fatalf("err=%v out=%s", err, &out)
				}
			})
		}
	}
	for raw, want := range map[string]string{
		"https://github.com/gkzz/renodash/issues/5": "https://github.com/gkzz/renodash/issues/5",
		"https://github.com/a(b)|[x]":               "https://github.com/a%28b%29%7C%5Bx%5D",
		"https://github.com/a%20b":                  "https://github.com/a%20b",
	} {
		got, err := linkURL(raw)
		if err != nil || got != want {
			t.Fatalf("%q => %q, %v; want %q", raw, got, err, want)
		}
	}
}

var errIO = errors.New("I/O failed")

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errIO }

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) { return len(p) - 1, nil }

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errIO }
func TestIOErrors(t *testing.T) {
	data, err := os.ReadFile("testdata/empty.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		w    io.Writer
		want error
	}{{failingWriter{}, errIO}, {shortWriter{}, io.ErrShortWrite}} {
		if err := RenderOffline(bytes.NewReader(data), tc.w); !errors.Is(err, tc.want) {
			t.Fatalf("want %v, got %v", tc.want, err)
		}
	}
	var out bytes.Buffer
	if err := RenderOffline(failingReader{}, &out); !errors.Is(err, errIO) || out.Len() != 0 {
		t.Fatalf("err=%v out=%s", err, &out)
	}
}
func TestSafe(t *testing.T) {
	got := safe("&amp; <b> \\ * _ [ ] | `\r\n\rnext\nline")
	want := "&amp;amp; &lt;b&gt; &#92; &#42; &#95; &#91; &#93; &#124; &#96; next line"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
