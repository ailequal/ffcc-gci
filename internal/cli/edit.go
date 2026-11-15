package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/ailequal/ffcc-gci/internal/save"
)

// editHelp is the part of the help text shared by every command that
// changes a save.
const editHelp = `  -slot  Save slot to edit, as numbered by "ffcc-gci info".
  -char  Character to edit: the name, or the number "ffcc-gci info" shows.
  -o     Write the result to this new file and leave the save untouched.

Without -o the save is edited in place, and the original is first copied
to a backup next to it, named <save.gci>.<date-time>.bak.

Close Dolphin before editing: it rewrites the save while a game runs.
`

// editFlags are the flags shared by every command that changes a save.
type editFlags struct {
	slot   *int
	char   *string
	output *string
}

func addEditFlags(fs *flag.FlagSet) *editFlags {
	return &editFlags{
		slot:   fs.Int("slot", 0, ""),
		char:   fs.String("char", "", ""),
		output: fs.String("o", "", ""),
	}
}

// requireFlags returns a usage error for the first of the named flags
// that was not given.
func requireFlags(fs *flag.FlagSet, names ...string) error {
	for _, name := range names {
		if !isSet(fs, name) {
			return usagef("%s needs -%s; run \"ffcc-gci %s -h\" for help", fs.Name(), name, fs.Name())
		}
	}
	return nil
}

// edit is one character of one slot of a save file, opened for editing.
type edit struct {
	file       *saveFile
	slotNumber int // as the user numbers slots, from 1
	slot       *save.Slot
	character  *save.Character
	output     string // -o, or "" to edit in place
}

// openEdit checks the shared flags, opens the save file and finds the
// slot and character to edit.
func openEdit(fs *flag.FlagSet, ef *editFlags, files []string) (*edit, error) {
	if err := requireFlags(fs, "slot", "char"); err != nil {
		return nil, err
	}
	if len(files) != 1 {
		return nil, usagef("%s needs exactly one save file, got %d", fs.Name(), len(files))
	}
	if *ef.slot < 1 || *ef.slot > save.SlotCount {
		return nil, usagef("-slot must be between 1 and %d, got %d", save.SlotCount, *ef.slot)
	}

	f, err := openSave(files[0])
	if err != nil {
		return nil, err
	}
	slot, err := f.data.Slot(*ef.slot - 1)
	if err != nil {
		return nil, fmt.Errorf("slot %d: %w", *ef.slot, err)
	}
	character, err := findCharacter(slot, *ef.char)
	if err != nil {
		return nil, fmt.Errorf("slot %d: %w", *ef.slot, err)
	}

	return &edit{file: f, slotNumber: *ef.slot, slot: slot, character: character, output: *ef.output}, nil
}

// who names the character being edited, for messages.
func (e *edit) who() string {
	return fmt.Sprintf("Slot %d, %s", e.slotNumber, e.character.Name())
}

// commit writes the edited slot to disk. allowed lists the parts of the
// decoded slot the edit was meant to change; if anything else differs
// from the original, nothing is written. summary is printed once the
// edit has passed that check.
func (e *edit) commit(stdout io.Writer, summary string, allowed ...save.Range) error {
	slotIndex := e.slotNumber - 1

	edited, err := save.Parse(e.file.data.Bytes())
	if err != nil {
		return err
	}
	if err := edited.SetSlot(slotIndex, e.slot); err != nil {
		return err
	}
	if err := save.VerifyEdit(e.file.data, edited, slotIndex, allowed...); err != nil {
		return fmt.Errorf("internal check failed, nothing written: %w", err)
	}
	content := e.file.withData(edited)

	fmt.Fprintln(stdout, summary)
	if e.output != "" {
		if err := writeNew(e.output, content, e.file.mode); err != nil {
			return fmt.Errorf("could not write the new file: %w", err)
		}
		fmt.Fprintf(stdout, "Written to %s; %s is unchanged.\n", e.output, e.file.path)
		return nil
	}

	backup, err := e.file.writeInPlace(content)
	if err != nil {
		if backup != "" {
			return fmt.Errorf("%w (the original is also saved at %s)", err, backup)
		}
		return err
	}
	fmt.Fprintf(stdout, "Saved %s\nBackup of the original: %s\n", e.file.path, backup)
	return nil
}

// findCharacter resolves a -char value: a number as shown by "info", or a
// name, compared without regard to case.
func findCharacter(slot *save.Slot, sel string) (*save.Character, error) {
	var existing []*save.Character
	for ci := range save.CharacterCount {
		if c, err := slot.Character(ci); err == nil && c.Exists() {
			existing = append(existing, c)
		}
	}
	if len(existing) == 0 {
		return nil, errors.New("the slot has no characters")
	}

	if number, err := strconv.Atoi(sel); err == nil {
		for _, c := range existing {
			if c.Index()+1 == number {
				return c, nil
			}
		}
		return nil, fmt.Errorf("there is no character number %d; available: %s", number, describe(existing))
	}

	var matches []*save.Character
	for _, c := range existing {
		if strings.EqualFold(c.Name(), sel) {
			matches = append(matches, c)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return nil, fmt.Errorf("there is no character named %q; available: %s", sel, describe(existing))
	default:
		return nil, fmt.Errorf("%d characters are named %q; pick one by number: %s", len(matches), sel, describe(matches))
	}
}

func describe(characters []*save.Character) string {
	parts := make([]string, len(characters))
	for i, c := range characters {
		parts[i] = fmt.Sprintf("%d (%s)", c.Index()+1, c.Name())
	}
	return strings.Join(parts, ", ")
}
