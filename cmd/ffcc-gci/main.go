// Ffcc-gci is a save editor for Final Fantasy Crystal Chronicles on the
// GameCube. It works on save files in Dolphin's .gci format and supports
// the European (PAL) version of the game only.
//
// It changes only what the game's own code is known to accept: a
// character's gil, artifacts and ordinary inventory items. Every value
// has a fixed range, and an edit never destroys the original save.
//
// Usage:
//
//	ffcc-gci <command> [arguments]
//
// Run "ffcc-gci help" for the list of commands and "ffcc-gci help
// <command>" for the flags and examples of one. The same reference is in
// docs/USAGE.md.
package main

import (
	"os"

	"github.com/ailequal/ffcc-gci/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
