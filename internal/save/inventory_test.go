package save_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/ailequal/ffcc-gci/internal/save"
	"github.com/ailequal/ffcc-gci/internal/save/savetest"
)

// dataWithItems builds a save whose only character holds the given items
// in inventory slots 0, 1, 2 and so on.
func dataWithItems(items ...int) []byte {
	return savetest.Data(&savetest.Slot{
		Town: "Tipa", Key: 11,
		Characters: []savetest.Character{{Name: "Alba", Gil: 10, Items: items}},
	})
}

func TestCheckItem(t *testing.T) {
	allowed := []int{293, 294, 351, 381, 392, 401, 493}
	for _, id := range allowed {
		if err := save.CheckItem(id); err != nil {
			t.Errorf("CheckItem(%d) = %v, want nil", id, err)
		}
	}

	refused := []int{
		-1, 0,
		1, 158, // equipment
		159, 231, 255, // artifacts and their placeholders
		256, 292, // magicite and commands
		352, 353, // Goddess Stat., Devil's Mask
		354,      // Extra 22
		355, 363, // seeds
		364, 367, // story items
		369, 370, // Ultimite, Dark Sphere
		380,      // Extra 33
		393, 400, // Flame Shard to Item Gil
		494, 500, // design 94 to design 100
		501, 65535,
	}
	for _, id := range refused {
		if err := save.CheckItem(id); !errors.Is(err, save.ErrOutOfRange) {
			t.Errorf("CheckItem(%d) = %v, want ErrOutOfRange", id, err)
		}
	}
}

// TestAddItemDoesWhatTheGameDoes adds an item and checks that the result
// is byte for byte the save that holds it in the next slot, with the
// count raised by one.
func TestAddItemDoesWhatTheGameDoes(t *testing.T) {
	for _, id := range []int{293, 294, 351, 381, 392, 401, 493} {
		before := mustParse(t, dataWithItems(1, 300))
		after := mustParse(t, dataWithItems(1, 300))
		s := mustSlot(t, after, 0)
		c := mustCharacter(t, s, 0)

		slot, err := c.AddItem(id)
		if err != nil {
			t.Fatalf("AddItem(%d): %v", id, err)
		}
		if slot != 2 {
			t.Errorf("AddItem(%d) used slot %d, want the first free slot, 2", id, slot)
		}
		if err := after.SetSlot(0, s); err != nil {
			t.Fatalf("SetSlot: %v", err)
		}

		if !bytes.Equal(after.Bytes(), dataWithItems(1, 300, id)) {
			t.Fatalf("AddItem(%d) did not produce the save that holds the item in slot 2", id)
		}
		if err := save.VerifyEdit(before, after, 0, c.ItemRange(slot), c.ItemCountRange()); err != nil {
			t.Errorf("VerifyEdit after AddItem(%d): %v", id, err)
		}
	}
}

func TestAddItemFillsAGap(t *testing.T) {
	// 0xFFFF in the list is an empty slot between two items.
	s := mustSlot(t, mustParse(t, dataWithItemsAndGap()), 0)
	c := mustCharacter(t, s, 0)

	slot, err := c.AddItem(300)
	if err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	if slot != 1 {
		t.Errorf("AddItem used slot %d, want the gap at slot 1", slot)
	}
	inv := c.Inventory()
	if inv[0] != 294 || inv[1] != 300 || inv[2] != 381 || c.ItemCount() != 3 {
		t.Errorf("inventory = %v, count %d", inv[:4], c.ItemCount())
	}
}

// dataWithItemsAndGap holds items in slots 0 and 2 and nothing in slot 1.
func dataWithItemsAndGap() []byte {
	return rewriteSlot(dataWithItems(294, 381, 381), func(plain []byte) {
		// Character 0, inventory slot 1 and the item count.
		rec := 0x14D0
		plain[rec+0x3C+2], plain[rec+0x3C+3] = 0xFF, 0xFF
		plain[rec+0x28], plain[rec+0x29] = 0, 2
	})
}

func TestAddItemAllowsDuplicatesExceptDesigns(t *testing.T) {
	s := mustSlot(t, mustParse(t, dataWithItems(293, 300, 381, 401)), 0)
	c := mustCharacter(t, s, 0)

	for _, id := range []int{293, 300, 381} {
		if _, err := c.AddItem(id); err != nil {
			t.Errorf("second copy of %d: AddItem = %v, want nil", id, err)
		}
	}
	if _, err := c.AddItem(401); !errors.Is(err, save.ErrAlreadyOwned) {
		t.Errorf("second copy of design 401: AddItem = %v, want ErrAlreadyOwned", err)
	}
	if c.ItemCount() != 7 {
		t.Errorf("item count = %d, want 7", c.ItemCount())
	}
}

func TestAddItemStopsAtTheGamesLimit(t *testing.T) {
	items := make([]int, 63)
	for i := range items {
		items[i] = 300
	}
	s := mustSlot(t, mustParse(t, dataWithItems(items...)), 0)
	c := mustCharacter(t, s, 0)

	if c.FreeSlots() != 1 {
		t.Fatalf("FreeSlots = %d, want 1", c.FreeSlots())
	}
	if slot, err := c.AddItem(381); err != nil || slot != 63 {
		t.Fatalf("AddItem into the last slot = %d, %v; want 63, nil", slot, err)
	}
	if _, err := c.AddItem(381); !errors.Is(err, save.ErrInventoryFull) {
		t.Errorf("AddItem into a full inventory = %v, want ErrInventoryFull", err)
	}
	if c.ItemCount() != save.InventorySize || c.FreeSlots() != 0 {
		t.Errorf("count = %d, free = %d, want 64 and 0", c.ItemCount(), c.FreeSlots())
	}
}

func TestAddItemRejects(t *testing.T) {
	s := mustSlot(t, mustParse(t, dataWithItems(294)), 0)
	c := mustCharacter(t, s, 0)

	for _, id := range []int{1, 159, 256, 352, 357, 364, 369, 400, 494} {
		if _, err := c.AddItem(id); !errors.Is(err, save.ErrOutOfRange) {
			t.Errorf("AddItem(%d) = %v, want ErrOutOfRange", id, err)
		}
	}
	if c.ItemCount() != 1 || c.FreeSlots() != 63 {
		t.Errorf("count = %d, free = %d after refused edits, want 1 and 63", c.ItemCount(), c.FreeSlots())
	}

	if _, err := mustCharacter(t, s, 3).AddItem(294); err == nil {
		t.Error("AddItem on an unused record succeeded, want an error")
	}
}

func TestAddItemRefusesAnInconsistentInventory(t *testing.T) {
	raw := rewriteSlot(dataWithItems(294, 381), func(plain []byte) {
		plain[0x14D0+0x29] = 5 // the count claims 5 items, 2 are held
	})

	c := mustCharacter(t, mustSlot(t, mustParse(t, raw), 0), 0)
	if _, err := c.AddItem(300); !errors.Is(err, save.ErrInventoryCount) {
		t.Errorf("AddItem = %v, want ErrInventoryCount", err)
	}
}

// rewriteSlot applies change to the decoded bytes of slot 0 of raw and
// returns the save data with the slot stored again under a fresh
// checksum.
func rewriteSlot(raw []byte, change func(plain []byte)) []byte {
	stored := raw[save.BannerSize : save.BannerSize+save.SlotSize]
	plain := save.Decode(stored)
	change(plain)
	sum := save.Checksum(plain)
	plain[0x1C], plain[0x1D], plain[0x1E], plain[0x1F] = byte(sum>>24), byte(sum>>16), byte(sum>>8), byte(sum)
	copy(stored, save.Encode(plain))
	return raw
}
