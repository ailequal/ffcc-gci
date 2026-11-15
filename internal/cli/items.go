package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/ailequal/ffcc-gci/internal/itemdb"
	"github.com/ailequal/ffcc-gci/internal/save"
)

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

// itemCatalog is the set of items that can be added to an inventory.
func itemCatalog() catalog {
	return catalog{
		noun:  "item",
		none:  `no item that can be added is named %q; run "ffcc-gci items" for the list`,
		ids:   addableIDs(),
		check: save.CheckItem,
		name:  itemName,
	}
}

// checkDesign applies the rules for adding a design to a character: one
// copy, and only a design for anyone or for the character's tribe. The
// forge screen of the game refuses to forge any other design.
func checkDesign(id, count int, character *save.Character) error {
	user, isDesign := itemdb.DesignFor(id)
	if !isDesign {
		return nil
	}
	desc, _ := itemdb.Description(id) // such as "Design for Selkies"
	what := strings.ToLower(desc[:1]) + desc[1:]
	name := itemName(id)

	switch {
	case user == itemdb.ForMen || user == itemdb.ForWomen:
		return fmt.Errorf("%s is a %s; designs for men or for women are not supported, nothing was written", name, what)
	case user != itemdb.ForAnyone && int(user) != int(character.Tribe()):
		return fmt.Errorf("%s is a %s and %s is a %s, who could not forge it; nothing was written",
			name, what, character.Name(), character.Tribe())
	case count != 1:
		return fmt.Errorf("%s is a design and can be added only once, not %d times; nothing was written", name, count)
	}
	return nil
}

func runItems(c *command, args []string, stdout, stderr io.Writer) error {
	filters, err := c.parse(c.flagSet(stderr), args, stdout)
	if err != nil {
		return err
	}
	if len(filters) > 1 {
		return usagef("items takes at most one text to search for, got %d", len(filters))
	}
	filter := strings.Join(filters, "")

	ids := itemCatalog().matching(filter)
	if len(ids) == 0 {
		return fmt.Errorf("no item that can be added has a name containing %q", filter)
	}

	group := itemdb.KindOther
	for i, id := range ids {
		if kind := itemdb.KindOf(id); kind != group {
			if i > 0 {
				fmt.Fprintln(stdout)
			}
			fmt.Fprintln(stdout, kind)
			group = kind
		}
		if group == itemdb.KindDesign {
			desc, _ := itemdb.Description(id)
			fmt.Fprintf(stdout, "  %d  %-15s  %s\n", id, itemName(id), desc)
		} else {
			fmt.Fprintf(stdout, "  %d  %s\n", id, itemName(id))
		}
	}
	return nil
}

func runAddItem(c *command, args []string, stdout, stderr io.Writer) error {
	fs := c.flagSet(stderr)
	ef := addEditFlags(fs)
	sel := fs.String("item", "", "")
	count := fs.Int("count", 1, "")

	files, err := c.parse(fs, args, stdout)
	if err != nil {
		return err
	}
	if err := requireFlags(fs, "slot", "char", "item"); err != nil {
		return err
	}
	id, err := itemCatalog().find(*sel)
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
	character := e.character

	if err := checkDesign(id, *count, character); err != nil {
		return err
	}
	if save.IsDesign(id) && character.HasItem(id) {
		fmt.Fprintf(stdout, "%s already has the design %s. A design is added only once. Nothing written.\n", e.who(), name)
		return nil
	}
	if free := character.FreeSlots(); *count > free {
		return fmt.Errorf("%s has %d free inventory slots of %d, not enough for %d; nothing was written",
			character.Name(), free, save.InventorySize, *count)
	}

	held := character.ItemCount()
	allowed := []save.Range{character.ItemCountRange()}
	for range *count {
		slot, err := character.AddItem(id)
		if err != nil {
			return fmt.Errorf("%w; nothing was written", err)
		}
		allowed = append(allowed, character.ItemRange(slot))
	}

	added := name
	if *count > 1 {
		added = fmt.Sprintf("%d x %s", *count, name)
	}
	summary := fmt.Sprintf("%s: added %s (items %d -> %d of %d)", e.who(), added, held, character.ItemCount(), save.InventorySize)
	return e.commit(stdout, summary, allowed...)
}
