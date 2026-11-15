package cli

import (
	"fmt"
	"io"

	"github.com/ailequal/ffcc-gci/internal/save"
)

func runSetGil(c *command, args []string, stdout, stderr io.Writer) error {
	fs := c.flagSet(stderr)
	ef := addEditFlags(fs)
	gil := fs.Int("gil", 0, "")

	files, err := c.parse(fs, args, stdout)
	if err != nil {
		return err
	}
	if err := requireFlags(fs, "slot", "char", "gil"); err != nil {
		return err
	}
	if err := save.GilLimit.Check(*gil); err != nil {
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
