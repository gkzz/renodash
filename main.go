package main

import (
	"os"

	"github.com/gkzz/renodash/internal/command"
)

func main() {
	os.Exit(command.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
