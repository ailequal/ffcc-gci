// Package cli implements the ffcc-gci command line: argument parsing,
// command dispatch and user-facing output.
//
// It holds no knowledge of the save format; that lives in the gci, save
// and itemdb packages.
//
// Each command is described once, in commands.go: its summary, flags,
// notes and examples. The help the program prints and the reference in
// docs/USAGE.md are both rendered from that description, so they cannot
// drift apart.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"runtime"
	"runtime/debug"
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

// usageError is a mistake in the command line, as opposed to a command
// that ran and failed.
type usageError struct {
	msg string
}

func (e *usageError) Error() string { return e.msg }

func usagef(format string, args ...any) error {
	return &usageError{msg: fmt.Sprintf(format, args...)}
}

// errFlagParse marks a flag error that has already been reported.
var errFlagParse = errors.New("invalid flags")

// Run executes the command line given in args (without the program name)
// and returns the process exit code. Normal output goes to stdout,
// diagnostics to stderr.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, overview())
		return ExitUsage
	}

	switch args[0] {
	case "help", "-h", "--help":
		return exitCode(runHelp(args[1:], stdout, stderr), stderr)
	case "version", "--version":
		fmt.Fprintln(stdout, versionLine())
		return ExitOK
	}

	if c := findCommand(commands, args[0]); c != nil {
		return exitCode(c.run(c, args[1:], stdout, stderr), stderr)
	}

	fmt.Fprintf(stderr, "ffcc-gci: unknown command %q\n\n", args[0])
	fmt.Fprint(stderr, overview())
	return ExitUsage
}

func findCommand(list []*command, name string) *command {
	for _, c := range list {
		if c.name == name {
			return c
		}
	}
	return nil
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

// versionLine is what "ffcc-gci version" prints: the program version, the
// Go toolchain and the platform. The release workflow checks this line on
// every platform it builds for.
func versionLine() string {
	return fmt.Sprintf("ffcc-gci %s (%s, %s/%s)", version(), runtime.Version(), runtime.GOOS, runtime.GOARCH)
}

// version returns the program version: the one set at build time, or the
// module version when the program was installed with "go install".
func version() string {
	if Version != "dev" {
		return Version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return Version
}

// flagSet returns a flag set for the command. The flag package reports a
// bad flag to stderr; the help text is printed by parse.
func (c *command) flagSet(stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(c.name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {}
	return fs
}

// parse parses args and returns the positional arguments. Unlike
// flag.Parse on its own, it accepts flags before and after them. For -h
// it prints the command's help to stdout and returns flag.ErrHelp.
func (c *command) parse(fs *flag.FlagSet, args []string, stdout io.Writer) ([]string, error) {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				fmt.Fprint(stdout, c.help())
				return nil, err
			}
			fmt.Fprintf(fs.Output(), "Run \"ffcc-gci help %s\" for usage.\n", c.name)
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

// requireFlags returns a usage error for the first of the named flags
// that was not given.
func requireFlags(fs *flag.FlagSet, names ...string) error {
	given := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) { given[f.Name] = true })
	for _, name := range names {
		if !given[name] {
			return usagef("%s needs -%s; run \"ffcc-gci help %s\" for usage", fs.Name(), name, fs.Name())
		}
	}
	return nil
}
