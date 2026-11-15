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

const itemsUsage = `List the items that add-item can add, by group.

Usage:
  ffcc-gci items [text]

With text, only items whose name contains it are listed.

Only ordinary items are listed: Phoenix Down, materials, food and designs.
Equipment, magicite, seeds, story items and unused entries of the game's
item table are left out for safety and cannot be added.
`

// addItemUsage is a format string; it takes the number of inventory
// slots.
const addItemUsage = `Add an item to a character's inventory.

Usage:
  ffcc-gci add-item -slot <1-4> -char <name or 1-8> -item <name> [-count <n>] [-o <new file>] <save.gci>

Flags:
  -item   The item to add: its name, a part of the name that matches only
          one item, or its item ID. "ffcc-gci items" lists what can be
          added; anything else is refused.
  -count  How many to add, 1 by default. Items do not stack: each one
          takes one of the character's %d inventory slots, and the
          command refuses to add more than there are free slots.
` + editHelp + `
Designs have two extra rules. A design is added only once, and only if it
is for anyone or for the character's own tribe. Designs for men or for
women are not supported.
`

// addableIDs returns the IDs of every item that can be added, in order.
func addableIDs() []int {
	var ids []int
	for _, l := range save.ItemLimits {
		for id := l.Min; id <= l.Max; id++ {
			ids = append(ids, id)
		}
	}
	return ids
}

func itemName(id int) string {
	name, _ := itemdb.Name(id)
	return name
}

// findItem resolves an -item value to the ID of an item that can be
// added: an item ID, an exact name, or a part of a name that matches a
// single item. Names are compared without regard to case.
func findItem(sel string) (int, error) {
	if id, err := strconv.Atoi(sel); err == nil {
		if err := save.CheckItem(id); err != nil {
			return 0, err
		}
		return id, nil
	}

	want := strings.ToLower(strings.TrimSpace(sel))
	if want == "" {
		return 0, errors.New("-item is empty")
	}
	var partial []int
	for _, id := range addableIDs() {
		name := strings.ToLower(itemName(id))
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
		return 0, fmt.Errorf("no item that can be added is named %q; run \"ffcc-gci items\" for the list", sel)
	default:
		names := make([]string, len(partial))
		for i, id := range partial {
			names[i] = itemName(id)
		}
		return 0, fmt.Errorf("%q matches %d items, be more specific: %s", sel, len(partial), strings.Join(names, ", "))
	}
}

// checkDesign applies the rules for adding a design to a character: one
// copy, and only a design for anyone or for the character's tribe.
func checkDesign(id, count int, character *save.Character) error {
	user, isDesign := itemdb.DesignFor(id)
	if !isDesign {
		return nil
	}
	desc, _ := itemdb.Description(id)
	name := itemName(id)

	switch {
	case user == itemdb.ForMen || user == itemdb.ForWomen:
		return fmt.Errorf("%s is a %s; designs for men or for women are not supported, nothing was written",
			name, strings.ToLower(desc))
	case user != itemdb.ForAnyone && int(user) != int(character.Tribe()):
		return fmt.Errorf("%s is a %s and %s is a %s, who could not forge it; nothing was written",
			name, strings.ToLower(desc[:1])+desc[1:], character.Name(), character.Tribe())
	case count != 1:
		return fmt.Errorf("%s is a design and can be added only once, not %d times; nothing was written", name, count)
	}
	return nil
}

func runItems(args []string, stdout, stderr io.Writer) error {
	fs := newFlagSet("items", stderr, itemsUsage)
	filters, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(filters) > 1 {
		return usagef("items takes at most one text to search for, got %d", len(filters))
	}
	filter := ""
	if len(filters) == 1 {
		filter = strings.ToLower(filters[0])
	}

	shown := 0
	group := itemdb.KindOther
	for _, id := range addableIDs() {
		name := itemName(id)
		if !strings.Contains(strings.ToLower(name), filter) {
			continue
		}
		if kind := itemdb.KindOf(id); kind != group {
			if shown > 0 {
				fmt.Fprintln(stdout)
			}
			fmt.Fprintf(stdout, "%s\n", kind)
			group = kind
		}
		if group == itemdb.KindDesign {
			desc, _ := itemdb.Description(id)
			fmt.Fprintf(stdout, "  %d  %-15s  %s\n", id, name, desc)
		} else {
			fmt.Fprintf(stdout, "  %d  %s\n", id, name)
		}
		shown++
	}

	if shown == 0 {
		return fmt.Errorf("no item that can be added has a name containing %q", filters[0])
	}
	return nil
}

func runAddItem(args []string, stdout, stderr io.Writer) error {
	fs := newFlagSet("add-item", stderr, fmt.Sprintf(addItemUsage, save.InventorySize))
	ef := addEditFlags(fs)
	sel := fs.String("item", "", "")
	count := fs.Int("count", 1, "")

	files, err := parse(fs, args)
	if err != nil {
		return err
	}
	if err := requireFlags(fs, "slot", "char", "item"); err != nil {
		return err
	}
	id, err := findItem(*sel)
	if err != nil {
		return usagef("%v; nothing was written", err)
	}
	if *count < 1 || *count > save.InventorySize {
		return usagef("-count must be between 1 and %d, got %d; nothing was written", save.InventorySize, *count)
	}
	name := itemName(id)

	e, err := openEdit(fs, ef, files)
	if err != nil {
		return err
	}
	c := e.character

	if err := checkDesign(id, *count, c); err != nil {
		return err
	}
	if save.IsDesign(id) && c.HasItem(id) {
		fmt.Fprintf(stdout, "%s already has the design %s. A design is added only once. Nothing written.\n", e.who(), name)
		return nil
	}
	if free := c.FreeSlots(); *count > free {
		return fmt.Errorf("%s has %d free inventory slots of %d, not enough for %d; nothing was written",
			c.Name(), free, save.InventorySize, *count)
	}

	held := c.ItemCount()
	allowed := []save.Range{c.ItemCountRange()}
	for range *count {
		slot, err := c.AddItem(id)
		if err != nil {
			return fmt.Errorf("%w; nothing was written", err)
		}
		allowed = append(allowed, c.ItemRange(slot))
	}

	summary := fmt.Sprintf("%s: added %s (items %d -> %d of %d)", e.who(), name, held, c.ItemCount(), save.InventorySize)
	if *count > 1 {
		summary = fmt.Sprintf("%s: added %d x %s (items %d -> %d of %d)", e.who(), *count, name, held, c.ItemCount(), save.InventorySize)
	}
	return e.commit(stdout, summary, allowed...)
}
