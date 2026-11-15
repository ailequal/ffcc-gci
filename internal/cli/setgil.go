package cli

import (
	"fmt"
	"io"

	"github.com/ailequal/ffcc-gci/internal/save"
)

// setGilUsage is a format string; it takes the lowest and highest gil.
const setGilUsage = `Set a character's gil.

Usage:
  ffcc-gci set-gil -slot <1-4> -char <name or 1-8> -gil <amount> [-o <new file>] <save.gci>

Flags:
  -gil   New amount of gil, %d to %d. This is the range the game itself
         keeps gil in; anything outside it is refused.
` + editHelp

func runSetGil(args []string, stdout, stderr io.Writer) error {
	limit := save.GilLimit
	fs := newFlagSet("set-gil", stderr, fmt.Sprintf(setGilUsage, limit.Min, limit.Max))
	ef := addEditFlags(fs)
	gil := fs.Int("gil", 0, "")

	files, err := parse(fs, args)
	if err != nil {
		return err
	}
	if err := requireFlags(fs, "slot", "char", "gil"); err != nil {
		return err
	}
	if err := limit.Check(*gil); err != nil {
		return usagef("%v; nothing was written", err)
	}

	e, err := openEdit(fs, ef, files)
	if err != nil {
		return err
	}

	old := e.character.Gil()
	if old == *gil {
		fmt.Fprintf(stdout, "%s: gil is already %d. Nothing written.\n", e.who(), old)
		return nil
	}
	if err := e.character.SetGil(*gil); err != nil {
		return err
	}

	summary := fmt.Sprintf("%s: gil %d -> %d", e.who(), old, *gil)
	return e.commit(stdout, summary, e.character.GilRange())
}
