package cli

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/ailequal/ffcc-gci/internal/itemdb"
	"github.com/ailequal/ffcc-gci/internal/save"
)

const infoUsage = `Show the slots and characters of a save file. Nothing is modified.

Usage:
  ffcc-gci info <save.gci>
`

// infoNote closes the output of info. It matters after an artifact has
// been granted: the artifact is listed, but the numbers are still the old
// ones until the game has saved again.
const infoNote = `
Hearts and stats are the numbers the game last stored in the save. The
game works them out again from tribe, equipment and artifacts when the
save is loaded.
`

func runInfo(args []string, stdout, stderr io.Writer) error {
	fs := newFlagSet("info", stderr, infoUsage)
	files, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(files) != 1 {
		return usagef("info needs exactly one save file, got %d", len(files))
	}

	f, err := openSave(files[0])
	if err != nil {
		return err
	}

	fmt.Fprintf(stdout, "%s: Final Fantasy Crystal Chronicles (PAL)\n", filepath.Base(f.path))
	for index := range save.SlotCount {
		fmt.Fprintln(stdout)
		slot, err := f.data.Slot(index)
		switch {
		case errors.Is(err, save.ErrEmptySlot):
			fmt.Fprintf(stdout, "Slot %d: empty\n", index+1)
		case err != nil:
			fmt.Fprintf(stdout, "Slot %d: not loadable (%v)\n", index+1, err)
		default:
			printSlot(stdout, index, slot)
		}
	}
	fmt.Fprint(stdout, infoNote)
	return nil
}

func printSlot(w io.Writer, index int, slot *save.Slot) {
	hours, minutes := slot.PlayTime()
	fmt.Fprintf(w, "Slot %d: town %s, play time %d:%02d\n", index+1, slot.TownName(), hours, minutes)

	for ci := range save.CharacterCount {
		c, err := slot.Character(ci)
		if err != nil || !c.Exists() {
			continue
		}

		party := ""
		if slot.InParty(ci) {
			party = ", in party"
		}
		fmt.Fprintf(w, "  %d. %s (%s%s)\n", ci+1, c.Name(), c.Tribe(), party)
		fmt.Fprintf(w, "     Gil        %d\n", c.Gil())
		fmt.Fprintf(w, "     Hearts     %s of %s\n", hearts(c.HP()), hearts(c.MaxHP()))
		fmt.Fprintf(w, "     Stats      strength %d, magic %d, defence %d\n", c.Strength(), c.Magic(), c.Defense())

		var items []int
		for _, id := range c.Inventory() {
			if id != save.EmptyItem {
				items = append(items, id)
			}
		}
		fmt.Fprintf(w, "     Items      %d of %d%s\n", len(items), save.InventorySize, nameList(items))

		artifacts := c.Artifacts()
		fmt.Fprintf(w, "     Artifacts  %d%s\n", len(artifacts), nameList(artifacts))
	}
}

// hearts formats hit points, which the save counts in half hearts.
func hearts(halves int) string {
	if halves%2 == 0 {
		return fmt.Sprint(halves / 2)
	}
	return fmt.Sprintf("%d.5", halves/2)
}

// nameList formats item IDs as ": name, name x2, ...", grouping repeated
// items in order of first appearance. It returns "" for no items.
func nameList(ids []int) string {
	if len(ids) == 0 {
		return ""
	}

	counts := make(map[int]int)
	var order []int
	for _, id := range ids {
		if counts[id] == 0 {
			order = append(order, id)
		}
		counts[id]++
	}

	parts := make([]string, len(order))
	for i, id := range order {
		name, ok := itemdb.Name(id)
		if !ok {
			name = fmt.Sprintf("unknown item %#x", id)
		}
		if n := counts[id]; n > 1 {
			name = fmt.Sprintf("%s x%d", name, n)
		}
		parts[i] = name
	}
	return ": " + strings.Join(parts, ", ")
}
