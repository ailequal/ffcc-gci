// Package cli implements the ffcc-gci command line: argument parsing,
// command dispatch and user-facing output.
//
// It holds no knowledge of the save format; that lives in the gci, save
// and itemdb packages.
package cli

import (
	"fmt"
	"io"
)

// Version is the program version. Release builds override it with
// -ldflags "-X github.com/ailequal/ffcc-gci/internal/cli.Version=...".
var Version = "dev"

// Exit codes returned by Run.
const (
	ExitOK    = 0 // success
	ExitError = 1 // the command ran and failed
	ExitUsage = 2 // the command line was invalid
)

const usage = `ffcc-gci - save editor for Final Fantasy Crystal Chronicles (GameCube)

Usage:
  ffcc-gci <command> [arguments]

Commands:
  help       Show this help
  version    Print the program version

No save-editing commands exist yet; see README.md for the planned set.
`

// Run executes the command line given in args (without the program name)
// and returns the process exit code. Normal output goes to stdout,
// diagnostics to stderr.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return ExitUsage
	}

	switch args[0] {
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return ExitOK
	case "version", "--version":
		fmt.Fprintf(stdout, "ffcc-gci %s\n", Version)
		return ExitOK
	default:
		fmt.Fprintf(stderr, "ffcc-gci: unknown command %q\n\n", args[0])
		fmt.Fprint(stderr, usage)
		return ExitUsage
	}
}
