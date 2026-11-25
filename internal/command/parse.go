package command

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"

	dashpkg "github.com/gkzz/renodash/internal/dashboard"
)

func executeParse(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("parse", flag.ContinueOnError)
	// Collect flag diagnostics ourselves so errors are printed only once.
	flags.SetOutput(io.Discard)
	format := flags.String("format", "json", "output format (json)")
	flags.Usage = func() {}
	if err := flags.Parse(args); err != nil {
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
	dashboard, err := dashpkg.Parse(stdin)
	if err != nil {
		return err
	}
	for _, control := range dashboard.UnknownControls {
		if _, err := fmt.Fprintf(stderr, "warning: unsupported Renovate control: %s\n", control); err != nil {
			return fmt.Errorf("write warning: %w", err)
		}
	}
	if dashboard.Items == nil {
		dashboard.Items = []dashpkg.Item{}
	}
	if dashboard.BulkActions == nil {
		dashboard.BulkActions = []dashpkg.BulkAction{}
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
