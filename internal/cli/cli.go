// Package cli implements the ffcc-gci command line: argument parsing,
// command dispatch and user-facing output.
//
// It holds no knowledge of the save format; that lives in the gci, save
// and itemdb packages.
package cli

import (
	"errors"
	"flag"
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

type command struct {
	name    string
	summary string
	run     func(args []string, stdout, stderr io.Writer) error
}

var commands = []command{
	{name: "info", summary: "Show the slots and characters of a save file", run: runInfo},
	{name: "set-gil", summary: "Set a character's gil", run: runSetGil},
	{name: "add-artifact", summary: "Grant a character an artifact", run: runAddArtifact},
	{name: "artifacts", summary: "List the artifacts that can be granted", run: runArtifacts},
	{name: "add-item", summary: "Add an item to a character's inventory", run: runAddItem},
	{name: "items", summary: "List the items that can be added", run: runItems},
	{name: "limits", summary: "Show every editable value and its allowed range", run: runLimits},
}

// usageError is a mistake in the command line, as opposed to a command
// that ran and failed.
type usageError struct {
	msg string
}

func (e *usageError) Error() string { return e.msg }

func usagef(format string, args ...any) error {
	return &usageError{msg: fmt.Sprintf(format, args...)}
}

// errFlagParse marks a flag error the flag package has already reported.
var errFlagParse = errors.New("invalid flags")

func printUsage(w io.Writer) {
	fmt.Fprint(w, `ffcc-gci - save editor for Final Fantasy Crystal Chronicles (GameCube, PAL)

Usage:
  ffcc-gci <command> [arguments]

Commands:
`)
	for _, c := range commands {
		fmt.Fprintf(w, "  %-12s  %s\n", c.name, c.summary)
	}
	fmt.Fprint(w, `  help          Show this help
  version       Print the program version

Run "ffcc-gci <command> -h" for the arguments of a command.
`)
}

// Run executes the command line given in args (without the program name)
// and returns the process exit code. Normal output goes to stdout,
// diagnostics to stderr.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return ExitUsage
	}

	switch args[0] {
	case "help", "-h", "--help":
		printUsage(stdout)
		return ExitOK
	case "version", "--version":
		fmt.Fprintf(stdout, "ffcc-gci %s\n", Version)
		return ExitOK
	}

	for _, c := range commands {
		if c.name == args[0] {
			return exitCode(c.run(args[1:], stdout, stderr), stderr)
		}
	}

	fmt.Fprintf(stderr, "ffcc-gci: unknown command %q\n\n", args[0])
	printUsage(stderr)
	return ExitUsage
}

func exitCode(err error, stderr io.Writer) int {
	var usage *usageError
	switch {
	case err == nil, errors.Is(err, flag.ErrHelp):
		return ExitOK
	case errors.Is(err, errFlagParse):
		return ExitUsage
	case errors.As(err, &usage):
		fmt.Fprintf(stderr, "ffcc-gci: %v\n", err)
		return ExitUsage
	default:
		fmt.Fprintf(stderr, "ffcc-gci: %v\n", err)
		return ExitError
	}
}

// newFlagSet returns a flag set for a command that reports errors to
// stderr and prints usage as its help text.
func newFlagSet(name string, stderr io.Writer, usage string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(fs.Output(), usage) }
	return fs
}

// parse parses args and returns the positional arguments. Unlike
// flag.Parse on its own, it accepts flags before and after them.
func parse(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil, err
			}
			return nil, errFlagParse
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return positional, nil
		}
		positional = append(positional, rest[0])
		args = rest[1:]
	}
}

// isSet reports whether the named flag was given on the command line.
func isSet(fs *flag.FlagSet, name string) bool {
	set := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			set = true
		}
	})
	return set
}
