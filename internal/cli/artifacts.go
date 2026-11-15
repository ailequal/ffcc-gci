package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ailequal/ffcc-gci/internal/itemdb"
	"github.com/ailequal/ffcc-gci/internal/save"
)

// statNote is printed after an artifact has been granted.
const statNote = "In the game the artifact counts at once: the game works out strength, magic, " +
	"defence, hearts and command slots from the artifacts by itself. \"ffcc-gci info\" will list " +
	"the artifact now, but its stat numbers are the ones the game last stored, so they stay the " +
	"same until the game saves."

// artifactIDs returns the item IDs of the artifacts that can be granted,
// in order.
func artifactIDs() []int {
	var ids []int
	for id := save.ArtifactLimit.Min; id <= save.ArtifactLimit.Max; id++ {
		ids = append(ids, id)
	}
	return ids
}

// artifactCatalog is the set of artifacts that can be granted.
func artifactCatalog() catalog {
	return catalog{
		noun:  "artifact",
		none:  `no artifact is named %q; run "ffcc-gci artifacts" for the list`,
		ids:   artifactIDs(),
		check: save.ArtifactLimit.Check,
		name:  itemName,
	}
}

func runArtifacts(c *command, args []string, stdout, stderr io.Writer) error {
	filters, err := c.parse(c.flagSet(stderr), args, stdout)
	if err != nil {
		return err
	}
	if len(filters) > 1 {
		return usagef("artifacts takes at most one text to search for, got %d", len(filters))
	}
	filter := strings.Join(filters, "")

	ids := artifactCatalog().matching(filter)
	if len(ids) == 0 {
		return fmt.Errorf("no artifact name contains %q", filter)
	}

	group := ""
	for i, id := range ids {
		if effect, _ := itemdb.ArtifactEffect(id); effect != group {
			if i > 0 {
				fmt.Fprintln(stdout)
			}
			fmt.Fprintln(stdout, effect)
			group = effect
		}
		desc, _ := itemdb.Description(id)
		fmt.Fprintf(stdout, "  %d  %-15s  %s\n", id, itemName(id), desc)
	}
	return nil
}

func runAddArtifact(c *command, args []string, stdout, stderr io.Writer) error {
	fs := c.flagSet(stderr)
	ef := addEditFlags(fs)
	sel := fs.String("artifact", "", "")

	files, err := c.parse(fs, args, stdout)
	if err != nil {
		return err
	}
	if err := requireFlags(fs, "slot", "char", "artifact"); err != nil {
		return err
	}
	id, err := artifactCatalog().find(*sel)
	if err != nil {
		return usagef("%v; nothing was written", err)
	}
	name := itemName(id)

	e, err := openEdit(fs, ef, files)
	if err != nil {
		return err
	}

	err = e.character.AddArtifact(id)
	if errors.Is(err, save.ErrAlreadyOwned) {
		fmt.Fprintf(stdout, "%s already has %s. An artifact cannot be held twice. Nothing written.\n", e.who(), name)
		return nil
	}
	if err != nil {
		return err
	}

	effect, _ := itemdb.ArtifactEffect(id)
	summary := fmt.Sprintf("%s: added artifact %s (%s)", e.who(), name, strings.ToLower(effect))
	if err := e.commit(stdout, summary, e.character.ArtifactRange(id)); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "\n%s\n", wrap(statNote, helpWidth, ""))
	return nil
}
