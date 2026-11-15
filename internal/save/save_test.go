package save_test

import (
	"bytes"
	"errors"
	"os"
	"slices"
	"testing"

	"github.com/ailequal/ffcc-gci/internal/gci"
	"github.com/ailequal/ffcc-gci/internal/save"
	"github.com/ailequal/ffcc-gci/internal/save/savetest"
)

func sample() []byte {
	return savetest.Data(
		&savetest.Slot{
			Town: "Tipa", Frames: 1500 * 75, Key: 186,
			Characters: []savetest.Character{
				{Name: "Alba", Tribe: 0, Gil: 200, Items: []int{0x01, 0x125, 0x125}},
				{Name: "Bruno", Tribe: 3, Gil: 346, Artifacts: []int{1, 24, 60, 69}},
			},
		},
		nil,
		&savetest.Slot{Town: "Tipa", Key: 7, Characters: []savetest.Character{{Name: "Alba", Gil: 5}}},
	)
}

func mustParse(t *testing.T, b []byte) *save.Data {
	t.Helper()
	d, err := save.Parse(b)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return d
}

func mustSlot(t *testing.T, d *save.Data, index int) *save.Slot {
	t.Helper()
	s, err := d.Slot(index)
	if err != nil {
		t.Fatalf("Slot(%d): %v", index, err)
	}
	return s
}

func mustCharacter(t *testing.T, s *save.Slot, index int) *save.Character {
	t.Helper()
	c, err := s.Character(index)
	if err != nil {
		t.Fatalf("Character(%d): %v", index, err)
	}
	return c
}

func TestCheckIdentity(t *testing.T) {
	if err := save.CheckIdentity("GCCP", "01", "FFCC"); err != nil {
		t.Errorf("PAL identity rejected: %v", err)
	}

	rejected := [][3]string{
		{"GCCE", "01", "FFCC"}, // USA
		{"GCCJ", "01", "FFCC"}, // Japan
		{"GCCP", "01", "OTHER"},
		{"GZLP", "01", "FFCC"},
	}
	for _, id := range rejected {
		if err := save.CheckIdentity(id[0], id[1], id[2]); !errors.Is(err, save.ErrUnsupported) {
			t.Errorf("CheckIdentity(%q) = %v, want ErrUnsupported", id, err)
		}
	}
}

func TestParseRejectsWrongSize(t *testing.T) {
	for _, size := range []int{0, save.DataSize - 1, save.DataSize + 1} {
		if _, err := save.Parse(make([]byte, size)); !errors.Is(err, save.ErrUnsupported) {
			t.Errorf("Parse(%d bytes) = %v, want ErrUnsupported", size, err)
		}
	}
}

func TestSlotStates(t *testing.T) {
	d := mustParse(t, sample())

	mustSlot(t, d, 0)
	mustSlot(t, d, 2)
	for _, index := range []int{1, 3} {
		if _, err := d.Slot(index); !errors.Is(err, save.ErrEmptySlot) {
			t.Errorf("Slot(%d) = %v, want ErrEmptySlot", index, err)
		}
	}
	for _, index := range []int{-1, save.SlotCount} {
		if _, err := d.Slot(index); err == nil {
			t.Errorf("Slot(%d) succeeded, want an error", index)
		}
	}
}

func TestSlotRejectsDamage(t *testing.T) {
	// Offsets are relative to the start of slot 0 on the card.
	tests := []struct {
		name string
		off  int
	}{
		{name: "maker tag", off: 0x00},
		{name: "version tag", off: 0x0C},
		{name: "region", off: 0x10},
		{name: "stored checksum", off: 0x1C},
		{name: "character data", off: 0x14D0 + 0xEC},
		{name: "last checksummed byte", off: 0x8BD0 - 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := sample()
			raw[save.BannerSize+tt.off] ^= 0x01

			if _, err := mustParse(t, raw).Slot(0); !errors.Is(err, save.ErrCorruptSlot) {
				t.Errorf("Slot(0) = %v, want ErrCorruptSlot", err)
			}
		})
	}
}

func TestSlotFields(t *testing.T) {
	s := mustSlot(t, mustParse(t, sample()), 0)

	if got := s.TownName(); got != "Tipa" {
		t.Errorf("TownName = %q, want %q", got, "Tipa")
	}
	if h, m := s.PlayTime(); h != 1 || m != 15 {
		t.Errorf("PlayTime = %d:%02d, want 1:15", h, m)
	}
	if !s.InParty(0) || !s.InParty(1) || s.InParty(2) {
		t.Errorf("InParty = %v %v %v, want true true false", s.InParty(0), s.InParty(1), s.InParty(2))
	}

	alba := mustCharacter(t, s, 0)
	if !alba.Exists() || alba.Name() != "Alba" || alba.Tribe() != save.Clavat || alba.Gil() != 200 {
		t.Errorf("character 0 = exists %v, %q, %v, %d gil", alba.Exists(), alba.Name(), alba.Tribe(), alba.Gil())
	}
	if alba.HP() != 8 || alba.MaxHP() != 8 || alba.Strength() != 10 || alba.Magic() != 11 || alba.Defense() != 12 {
		t.Errorf("character 0 stats = hp %d/%d str %d mag %d def %d",
			alba.HP(), alba.MaxHP(), alba.Strength(), alba.Magic(), alba.Defense())
	}
	inv := alba.Inventory()
	if alba.ItemCount() != 3 || inv[0] != 0x01 || inv[1] != 0x125 || inv[2] != 0x125 || inv[3] != save.EmptyItem {
		t.Errorf("character 0 inventory = count %d, %v", alba.ItemCount(), inv[:4])
	}

	bruno := mustCharacter(t, s, 1)
	if bruno.Tribe() != save.Selkie {
		t.Errorf("character 1 tribe = %v, want Selkie", bruno.Tribe())
	}
	// savetest sets flags 1, 24, 60 and 69: Maneater, Mage Masher, Moogle
	// Pocket and Earth Pendant.
	if got, want := bruno.Artifacts(), []int{0xA0, 0xB7, 0xDB, 0xE4}; !slices.Equal(got, want) {
		t.Errorf("character 1 artifacts = %v, want %v", got, want)
	}

	if c := mustCharacter(t, s, 2); c.Exists() {
		t.Error("character 2 exists, want an unused record")
	}
	if _, err := s.Character(save.CharacterCount); err == nil {
		t.Error("Character(8) succeeded, want an error")
	}
}

func TestSetSlotWithoutChangesIsIdentity(t *testing.T) {
	raw := sample()
	d := mustParse(t, raw)

	for _, index := range []int{0, 2} {
		if err := d.SetSlot(index, mustSlot(t, d, index)); err != nil {
			t.Fatalf("SetSlot(%d): %v", index, err)
		}
	}

	if !bytes.Equal(d.Bytes(), raw) {
		t.Error("storing unchanged slots changed the save data")
	}
}

func TestSetGil(t *testing.T) {
	before := mustParse(t, sample())
	after := mustParse(t, sample())

	s := mustSlot(t, after, 0)
	bruno := mustCharacter(t, s, 1)
	if err := bruno.SetGil(save.MaxGil); err != nil {
		t.Fatalf("SetGil: %v", err)
	}
	if err := after.SetSlot(0, s); err != nil {
		t.Fatalf("SetSlot: %v", err)
	}

	// The edit must survive a reload, leave everything else alone and
	// pass the same verification the command runs before writing.
	reloaded := mustSlot(t, mustParse(t, after.Bytes()), 0)
	if got := mustCharacter(t, reloaded, 1).Gil(); got != save.MaxGil {
		t.Errorf("gil after reload = %d, want %d", got, save.MaxGil)
	}
	if got := mustCharacter(t, reloaded, 0).Gil(); got != 200 {
		t.Errorf("other character's gil = %d, want 200", got)
	}
	if err := save.VerifyEdit(before, after, 0, bruno.GilRange()); err != nil {
		t.Errorf("VerifyEdit: %v", err)
	}
}

func TestSetGilRejects(t *testing.T) {
	s := mustSlot(t, mustParse(t, sample()), 0)

	alba := mustCharacter(t, s, 0)
	for _, gil := range []int{-1, save.MaxGil + 1} {
		if err := alba.SetGil(gil); err == nil {
			t.Errorf("SetGil(%d) succeeded, want an error", gil)
		}
	}
	if alba.Gil() != 200 {
		t.Errorf("gil = %d after rejected edits, want 200", alba.Gil())
	}

	if err := mustCharacter(t, s, 5).SetGil(100); err == nil {
		t.Error("SetGil on an unused record succeeded, want an error")
	}
}

func TestVerifyEditCatchesStrayChanges(t *testing.T) {
	edit := func(t *testing.T, change func(d *save.Data, s *save.Slot)) (before, after *save.Data, gil save.Range) {
		t.Helper()
		before, after = mustParse(t, sample()), mustParse(t, sample())
		s := mustSlot(t, after, 0)
		c := mustCharacter(t, s, 0)
		if err := c.SetGil(999); err != nil {
			t.Fatalf("SetGil: %v", err)
		}
		change(after, s)
		return before, after, c.GilRange()
	}

	t.Run("second field in the same slot", func(t *testing.T) {
		before, after, gil := edit(t, func(d *save.Data, s *save.Slot) {
			if err := mustCharacter(t, s, 1).SetGil(1); err != nil {
				t.Fatalf("SetGil: %v", err)
			}
			if err := d.SetSlot(0, s); err != nil {
				t.Fatalf("SetSlot: %v", err)
			}
		})
		if err := save.VerifyEdit(before, after, 0, gil); err == nil {
			t.Error("VerifyEdit accepted a change outside the allowed range")
		}
	})

	t.Run("another slot", func(t *testing.T) {
		before, after, gil := edit(t, func(d *save.Data, s *save.Slot) {
			if err := d.SetSlot(0, s); err != nil {
				t.Fatalf("SetSlot: %v", err)
			}
			other := mustSlot(t, d, 2)
			if err := mustCharacter(t, other, 0).SetGil(77); err != nil {
				t.Fatalf("SetGil: %v", err)
			}
			if err := d.SetSlot(2, other); err != nil {
				t.Fatalf("SetSlot: %v", err)
			}
		})
		if err := save.VerifyEdit(before, after, 0, gil); err == nil {
			t.Error("VerifyEdit accepted a change in another slot")
		}
	})
}

// TestRealSave runs against a real PAL save when one is present. Real
// saves are not kept in the repository, so the test is skipped without
// one. Put a save at testdata/pal.gci in the repository root to enable it.
func TestRealSave(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/pal.gci")
	if errors.Is(err, os.ErrNotExist) {
		t.Skip("no real save at testdata/pal.gci")
	}
	if err != nil {
		t.Fatal(err)
	}

	file, err := gci.Parse(raw)
	if err != nil {
		t.Fatalf("gci.Parse: %v", err)
	}
	if err := save.CheckIdentity(file.GameCode(), file.MakerCode(), file.FileName()); err != nil {
		t.Fatalf("CheckIdentity: %v", err)
	}
	d := mustParse(t, file.Data)

	valid := 0
	for index := range save.SlotCount {
		s, err := d.Slot(index)
		if errors.Is(err, save.ErrEmptySlot) {
			continue
		}
		if err != nil {
			t.Errorf("Slot(%d): %v", index, err)
			continue
		}
		valid++

		// Storing a slot back unchanged must not alter a single byte.
		if err := d.SetSlot(index, s); err != nil {
			t.Errorf("SetSlot(%d): %v", index, err)
		}

		// The recorded item count must match the inventory, as the game
		// expects on load.
		for ci := range save.CharacterCount {
			c := mustCharacter(t, s, ci)
			if !c.Exists() {
				continue
			}
			used := 0
			for _, id := range c.Inventory() {
				if id != save.EmptyItem {
					used++
				}
			}
			if used != c.ItemCount() {
				t.Errorf("slot %d character %d: %d items in inventory, count field says %d", index, ci, used, c.ItemCount())
			}
		}
	}

	if valid == 0 {
		t.Error("the real save has no valid slot")
	}
	if !bytes.Equal(d.Bytes(), file.Data) {
		t.Error("storing unchanged slots changed the real save")
	}
}
