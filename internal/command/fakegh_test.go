package command

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestMain also serves the copied test executable as an offline gh replacement.
// Only the gh basename enters helper mode; ordinary tests keep the usual runner.
func TestMain(m *testing.M) {
	if strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe") == "gh" {
		switch os.Getenv("COMMAND_FAKE_GH_MODE") {
		case "arguments":
			for _, arg := range os.Args[1:] {
				fmt.Fprintln(os.Stdout, arg)
			}
		case "output":
			fmt.Fprint(os.Stdout, os.Getenv("COMMAND_FAKE_GH_PAYLOAD"))
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
