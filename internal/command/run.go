package command

import (
	"errors"
	"fmt"
	"io"
)

func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	return runWithClient(newIssueHTTPClient(), args, stdin, stdout, stderr)
}

func runWithClient(client githubClient, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if err := execute(client, args, stdin, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

func execute(client githubClient, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		_, err := fmt.Fprintln(stdout, "Usage: renodash parse [--format json]\n       renodash get [--with-pr-metadata [--allow-partial]] [--stdin | OWNER/REPO#ISSUE_NUMBER]\n       renodash version\n\nParse standard input or fetch and parse a GitHub issue.\nUse version to show build information.")
		return err
	}
	if len(args) == 0 {
		return errors.New("expected subcommand: parse, get, or version (use --help for usage)")
	}
	if args[0] == "get" {
		return executeGetInput(client, args[1:], stdin, stdout, stderr)
	}
	if args[0] == "version" {
		if len(args) == 2 && (args[1] == "--help" || args[1] == "-h") {
			_, err := fmt.Fprintln(stdout, "Usage: renodash version\n\nShow version and optional build information.")
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
	return executeParse(args[1:], stdin, stdout, stderr)
}
