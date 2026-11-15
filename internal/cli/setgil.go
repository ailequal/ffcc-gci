package cli

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/ailequal/ffcc-gci/internal/save"
)

// setGilUsage is a format string; it takes the lowest and highest gil.
const setGilUsage = `Set a character's gil.

Usage:
  ffcc-gci set-gil -slot <1-4> -char <name or 1-8> -gil <amount> [-o <new file>] <save.gci>

Flags:
  -slot  Save slot to edit, as numbered by "ffcc-gci info".
  -char  Character to edit: the name, or the number "ffcc-gci info" shows.
  -gil   New amount of gil, %d to %d. This is the range the game itself
         keeps gil in; anything outside it is refused.
  -o     Write the result to this new file and leave the save untouched.

Without -o the save is edited in place, and the original is first copied
to a backup next to it, named <save.gci>.<date-time>.bak.

Close Dolphin before editing: it rewrites the save while a game runs.
`

func runSetGil(args []string, stdout, stderr io.Writer) error {
	limit := save.GilLimit
	fs := newFlagSet("set-gil", stderr, fmt.Sprintf(setGilUsage, limit.Min, limit.Max))
	slotNumber := fs.Int("slot", 0, "")
	charSel := fs.String("char", "", "")
	gil := fs.Int("gil", 0, "")
	output := fs.String("o", "", "")

	files, err := parse(fs, args)
	if err != nil {
		return err
	}
	for _, name := range []string{"slot", "char", "gil"} {
		if !isSet(fs, name) {
			return usagef("set-gil needs -%s; run \"ffcc-gci set-gil -h\" for help", name)
		}
	}
	if len(files) != 1 {
		return usagef("set-gil needs exactly one save file, got %d", len(files))
	}
	if *slotNumber < 1 || *slotNumber > save.SlotCount {
		return usagef("-slot must be between 1 and %d, got %d", save.SlotCount, *slotNumber)
	}
	if err := limit.Check(*gil); err != nil {
		return usagef("%v; nothing was written", err)
	}
	slotIndex := *slotNumber - 1

	f, err := openSave(files[0])
	if err != nil {
		return err
	}
	slot, err := f.data.Slot(slotIndex)
	if err != nil {
		return fmt.Errorf("slot %d: %w", *slotNumber, err)
	}
	character, err := findCharacter(slot, *charSel)
	if err != nil {
		return fmt.Errorf("slot %d: %w", *slotNumber, err)
	}

	old := character.Gil()
	if old == *gil {
		fmt.Fprintf(stdout, "Slot %d, %s: gil is already %d. Nothing written.\n", *slotNumber, character.Name(), old)
		return nil
	}

	if err := character.SetGil(*gil); err != nil {
		return err
	}
	edited, err := save.Parse(f.data.Bytes())
	if err != nil {
		return err
	}
	if err := edited.SetSlot(slotIndex, slot); err != nil {
		return err
	}
	if err := save.VerifyEdit(f.data, edited, slotIndex, character.GilRange()); err != nil {
		return fmt.Errorf("internal check failed, nothing written: %w", err)
	}
	content := f.withData(edited)

	fmt.Fprintf(stdout, "Slot %d, %s: gil %d -> %d\n", *slotNumber, character.Name(), old, *gil)
	if *output != "" {
		if err := writeNew(*output, content, f.mode); err != nil {
			return fmt.Errorf("could not write the new file: %w", err)
		}
		fmt.Fprintf(stdout, "Written to %s; %s is unchanged.\n", *output, f.path)
		return nil
	}

	backup, err := f.writeInPlace(content)
	if err != nil {
		if backup != "" {
			return fmt.Errorf("%w (the original is also saved at %s)", err, backup)
		}
		return err
	}
	fmt.Fprintf(stdout, "Saved %s\nBackup of the original: %s\n", f.path, backup)
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
