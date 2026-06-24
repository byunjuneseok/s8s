// Command s8s is a terminal UI for viewing securities accounts and trading.
package main

import (
	"fmt"
	"io"
	"os"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

const usageText = `s8s — a terminal UI for securities accounts and trading.

Usage:
  s8s version      print the version
  s8s help         print this help
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run executes the CLI and returns the process exit code. It is kept separate
// from main so it can be exercised in tests.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = io.WriteString(stdout, usageText)
		return 0
	}

	switch args[0] {
	case "version", "--version", "-v":
		_, _ = fmt.Fprintf(stdout, "s8s %s\n", version)
		return 0
	case "help", "--help", "-h":
		_, _ = io.WriteString(stdout, usageText)
		return 0
	default:
		_, _ = fmt.Fprintf(stderr, "s8s: unknown command %q\n\n", args[0])
		_, _ = io.WriteString(stderr, usageText)
		return 2
	}
}
