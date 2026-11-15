package cli

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/ailequal/ffcc-gci/internal/itemdb"
	"github.com/ailequal/ffcc-gci/internal/save"
)

const artifactsUsage = `List the artifacts that add-artifact can grant, grouped by what they improve.

Usage:
  ffcc-gci artifacts [text]

With text, only artifacts whose name contains it are listed.
`

// addArtifactUsage is a format string; it takes the lowest and highest
// artifact item ID.
const addArtifactUsage = `Grant a character an artifact.

Usage:
  ffcc-gci add-artifact -slot <1-4> -char <name or 1-8> -artifact <name> [-o <new file>] <save.gci>

Flags:
  -artifact  The artifact to grant: its name, a part of the name that
             matches only one artifact, or its item ID (%d to %d).
             "ffcc-gci artifacts" lists them all.
` + editHelp + `
An artifact is either owned or not, so it cannot be granted twice: if the
character already has it, nothing is written.

The game works out strength, magic, defence, hearts and command slots from
the artifacts by itself, so the new values are there as soon as the save
is loaded. "ffcc-gci info" reads the numbers the game last stored, and
keeps showing the old ones until the game has saved again.
`

// statNote is printed after an artifact has been granted.
const statNote = `
In the game the artifact counts at once: the game works out strength,
magic, defence, hearts and command slots from the artifacts by itself.
"ffcc-gci info" will list the artifact now, but its stat numbers are the
ones the game last stored, so they stay the same until the game saves.
`

// artifactName returns the name of an artifact in the grantable range.
func artifactName(id int) string {
	name, _ := itemdb.Name(id)
	return name
}

// findArtifact resolves an -artifact value to an item ID: an item ID, an
// exact name, or a part of a name that matches a single artifact. Names
// are compared without regard to case.
func findArtifact(sel string) (int, error) {
	limit := save.ArtifactLimit

	if id, err := strconv.Atoi(sel); err == nil {
		if err := limit.Check(id); err != nil {
			return 0, err
		}
		return id, nil
	}

	want := strings.ToLower(strings.TrimSpace(sel))
	if want == "" {
		return 0, errors.New("-artifact is empty")
	}
	var partial []int
	for id := limit.Min; id <= limit.Max; id++ {
		name := strings.ToLower(artifactName(id))
		if name == want {
			return id, nil
		}
		if strings.Contains(name, want) {
			partial = append(partial, id)
		}
	}

	switch len(partial) {
	case 1:
		return partial[0], nil
	case 0:
		return 0, fmt.Errorf("no artifact is named %q; run \"ffcc-gci artifacts\" for the list", sel)
	default:
		names := make([]string, len(partial))
		for i, id := range partial {
			names[i] = artifactName(id)
		}
		return 0, fmt.Errorf("%q matches %d artifacts, be more specific: %s", sel, len(partial), strings.Join(names, ", "))
	}
}

func runArtifacts(args []string, stdout, stderr io.Writer) error {
	fs := newFlagSet("artifacts", stderr, artifactsUsage)
	filters, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(filters) > 1 {
		return usagef("artifacts takes at most one text to search for, got %d", len(filters))
	}
	filter := ""
	if len(filters) == 1 {
		filter = strings.ToLower(filters[0])
	}

	limit := save.ArtifactLimit
	shown, group := 0, ""
	for id := limit.Min; id <= limit.Max; id++ {
		name := artifactName(id)
		if !strings.Contains(strings.ToLower(name), filter) {
			continue
		}
		if effect, _ := itemdb.ArtifactEffect(id); effect != group {
			if shown > 0 {
				fmt.Fprintln(stdout)
			}
			fmt.Fprintf(stdout, "%s\n", effect)
			group = effect
		}
		desc, _ := itemdb.Description(id)
		fmt.Fprintf(stdout, "  %d  %-15s  %s\n", id, name, desc)
		shown++
	}

	if shown == 0 {
		return fmt.Errorf("no artifact name contains %q", filters[0])
	}
	return nil
}

func runAddArtifact(args []string, stdout, stderr io.Writer) error {
	limit := save.ArtifactLimit
	fs := newFlagSet("add-artifact", stderr, fmt.Sprintf(addArtifactUsage, limit.Min, limit.Max))
	ef := addEditFlags(fs)
	sel := fs.String("artifact", "", "")

	files, err := parse(fs, args)
	if err != nil {
		return err
	}
	if err := requireFlags(fs, "slot", "char", "artifact"); err != nil {
		return err
	}
	id, err := findArtifact(*sel)
	if err != nil {
		return usagef("%v; nothing was written", err)
	}
	name := artifactName(id)

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
	fmt.Fprint(stdout, statNote)
	return nil
}
