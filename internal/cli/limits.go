package cli

import (
	"fmt"
	"io"

	"github.com/ailequal/ffcc-gci/internal/save"
)

const limitsUsage = `Show every value ffcc-gci can set and the range it accepts.

Usage:
  ffcc-gci limits
`

func runLimits(args []string, stdout, stderr io.Writer) error {
	fs := newFlagSet("limits", stderr, limitsUsage)
	extra, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(extra) != 0 {
		return usagef("limits takes no arguments")
	}

	fmt.Fprint(stdout, `Every value ffcc-gci can set, with the range it accepts. A value outside
its range is refused and nothing is written. Each range is the one the game
itself enforces, as shown by the FFCC decompilation.

`)
	for _, l := range save.Limits() {
		fmt.Fprintf(stdout, "%s: %d to %d\n  %s\n\n", l.Name, l.Min, l.Max, l.Source)
	}
	fmt.Fprint(stdout, `Artifacts are chosen by name; "ffcc-gci artifacts" lists them with their
item IDs.

`)
	fmt.Fprintf(stdout, `What an edit can target:

slot: 1 to %d
  The slot must hold a save the game would load. Empty and damaged slots
  are refused.

character: 1 to %d, or the character's name
  The character must exist in the slot. Characters are never created.
`, save.SlotCount, save.CharacterCount)
	return nil
}
