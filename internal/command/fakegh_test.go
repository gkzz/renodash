package command

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

// TestMain also serves the copied test executable as an offline gh replacement.
// Only the gh basename enters helper mode; ordinary tests keep the usual runner.
func TestMain(m *testing.M) {
	if strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe") == "gh" {
		switch os.Getenv("COMMAND_FAKE_GH_MODE") {
		case "offline":
			if err := serveGH(os.Args[1:]); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		case "auth-token":
			if !slices.Equal(os.Args[1:], []string{"auth", "token", "--hostname", "github.com"}) {
				os.Exit(2)
			}
			fmt.Fprint(os.Stdout, os.Getenv("COMMAND_FAKE_GH_PAYLOAD"))
		case "pages":
			var pages []string
			if err := json.Unmarshal([]byte(os.Getenv("COMMAND_FAKE_GH_PAYLOAD")), &pages); err != nil {
				os.Exit(2)
			}
			index := 0
			for _, arg := range os.Args[1:] {
				if arg == "cursor=next" {
					index = 1
				}
			}
			if index >= len(pages) {
				os.Exit(2)
			}
			if path := os.Getenv("COMMAND_FAKE_GH_LOG"); path != "" {
				f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
				if err != nil {
					os.Exit(2)
				}
				if err = json.NewEncoder(f).Encode(os.Args[1:]); err != nil {
					os.Exit(2)
				}
				f.Close()
			}
			fmt.Fprint(os.Stdout, pages[index])
		case "output":
			fmt.Fprint(os.Stdout, os.Getenv("COMMAND_FAKE_GH_PAYLOAD"))
		case "http-failure":
			fmt.Fprint(os.Stdout, os.Getenv("COMMAND_FAKE_GH_PAYLOAD"))
			fmt.Fprintln(os.Stderr, "sensitive diagnostic")
			os.Exit(1)
		case "failure":
			fmt.Fprintln(os.Stderr, os.Getenv("COMMAND_FAKE_GH_PAYLOAD"))
			os.Exit(1)
		case "wait":
			conn, err := net.DialTimeout("tcp", os.Getenv("COMMAND_FAKE_GH_PAYLOAD"), 5*time.Second)
			if err != nil {
				os.Exit(2)
			}
			// A deadline and parent-side connection cleanup also bound failure paths.
			if err := conn.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
				os.Exit(2)
			}
			fmt.Fprint(os.Stdout, "partial response")
			fmt.Fprint(os.Stderr, "HTTP 401 token=secret")
			if _, err := io.WriteString(conn, "ready\n"); err != nil {
				os.Exit(2)
			}
			// Remain running until killed or the parent closes the connection.
			_, _ = io.Copy(io.Discard, conn)
			_ = conn.Close()
		default:
			fmt.Fprintln(os.Stderr, "unsupported command fake gh mode")
			os.Exit(2)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func installFakeGH(t *testing.T, mode, payload string) {
	t.Helper()
	dir := t.TempDir()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	// The production client looks up "gh" through PATH. Unix needs that exact
	// filename; Windows discovers gh.exe through PATHEXT.
	name := "gh"
	if runtime.GOOS == "windows" {
		name = "gh.exe"
		t.Setenv("PATHEXT", ".EXE")
	}
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("COMMAND_FAKE_GH_MODE", mode)
	t.Setenv("COMMAND_FAKE_GH_PAYLOAD", payload)
}
