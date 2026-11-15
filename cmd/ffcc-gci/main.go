// Command ffcc-gci inspects and edits Final Fantasy Crystal Chronicles
// (GameCube) save files stored in Dolphin's .gci format.
package main

import (
	"os"

	"ffcc-gci/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
