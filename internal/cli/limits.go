package cli

import (
	"fmt"
	"io"

	"github.com/ailequal/ffcc-gci/internal/save"
)

// limitsIntro and limitsOutro frame the output of limits.
const (
	limitsIntro = "Every value ffcc-gci can set, with the range it accepts. A value outside its " +
		"range is refused and nothing is written. Each range is the one the game itself " +
		"enforces, as shown by the FFCC decompilation."
	limitsOutro = "Artifacts and items are chosen by name. \"ffcc-gci artifacts\" and \"ffcc-gci " +
		"items\" list the ones that are accepted, with their item IDs. Everything else in the " +
		"game's item table is refused for safety: equipment, magicite, seeds, story items and " +
		"unused entries."
)

func runLimits(c *command, args []string, stdout, stderr io.Writer) error {
	extra, err := c.parse(c.flagSet(stderr), args, stdout)
	if err != nil {
		return err
	}
	if len(extra) != 0 {
		return usagef("limits takes no arguments")
	}

	fmt.Fprintf(stdout, "%s\n\n", wrap(limitsIntro, helpWidth, ""))
	for _, l := range save.Limits() {
		fmt.Fprintf(stdout, "%s: %s\n%s\n\n", l.Name, rangeText(l), wrap(l.Source, helpWidth, "  "))
	}
	fmt.Fprintf(stdout, "slot: 1 to %d\n%s\n\n", save.SlotCount,
		wrap("The slot must hold a save the game would load. Empty and damaged slots are refused.", helpWidth, "  "))
	fmt.Fprintf(stdout, "character: 1 to %d, or the character's name\n%s\n\n", save.CharacterCount,
		wrap("The character must exist in the slot. Characters are never created.", helpWidth, "  "))
	fmt.Fprintf(stdout, "%s\n", wrap(limitsOutro, helpWidth, ""))
	return nil
}
