// fake-actions-summary renders saved fixture JSON with the real summary renderer.
// It is an internal fixture tool, not a public renodash command.
package main

import (
	"fmt"
	"os"

	"github.com/gkzz/renodash/internal/summary"
)

func main() {
	if len(os.Args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: fake-actions-summary < renodash.json")
		os.Exit(1)
	}
	if err := summary.RenderOffline(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
