package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/gkzz/renodash"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if err := execute(args, stdin, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

func execute(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		_, err := fmt.Fprintln(stdout, "Usage: renodash parse [--format json]\n       renodash get OWNER/REPO#ISSUE_NUMBER\n       renodash version\n\nParse standard input or fetch and parse a GitHub issue.\nUse version to show build information.")
		return err
	}
	if len(args) == 0 {
		return errors.New("expected subcommand: parse, get, or version (use --help for usage)")
	}
	if args[0] == "get" {
		return executeGet(args[1:], stdout, stderr)
	}
	if args[0] == "version" {
		if len(args) == 2 && (args[1] == "--help" || args[1] == "-h") {
			_, err := fmt.Fprintln(stdout, "Usage: renodash version\n\nShow version and build commit.")
			return err
		}
		if len(args) != 1 {
			return fmt.Errorf("unexpected argument: %s", args[1])
		}
		return printVersion(stdout)
	}
	if args[0] != "parse" {
		return fmt.Errorf("unknown subcommand: %s", args[0])
	}
	flags := flag.NewFlagSet("parse", flag.ContinueOnError)
	// Collect flag diagnostics ourselves so errors are printed only once.
	flags.SetOutput(io.Discard)
	format := flags.String("format", "json", "output format (json)")
	flags.Usage = func() {}
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, err = fmt.Fprintln(stdout, "Usage: renodash parse [--format json]\n\nRead Markdown from standard input.\n\nOptions:\n  --format string\n        output format (default json)")
			return err
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected argument: %s", flags.Arg(0))
	}
	if *format != "json" {
		return fmt.Errorf("unsupported output format: %s", *format)
	}
	dashboard, err := renodash.Parse(stdin)
	if err != nil {
		return err
	}
	for _, control := range dashboard.UnknownControls {
		if _, err := fmt.Fprintf(stderr, "warning: unsupported Renovate control: %s\n", control); err != nil {
			return fmt.Errorf("write warning: %w", err)
		}
	}
	if dashboard.Items == nil {
		dashboard.Items = []renodash.Item{}
	}
	if dashboard.BulkActions == nil {
		dashboard.BulkActions = []renodash.BulkAction{}
	}
	if dashboard.UnknownControls == nil {
		dashboard.UnknownControls = []string{}
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(dashboard); err != nil {
		return fmt.Errorf("write JSON: %w", err)
	}
	return nil
}
