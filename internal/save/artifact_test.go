package save_test

import (
	"bytes"
	"errors"
	"slices"
	"testing"

	"github.com/ailequal/ffcc-gci/internal/save"
	"github.com/ailequal/ffcc-gci/internal/save/savetest"
)

// dataWithArtifacts builds a save whose only character owns the given
// artifact flags. savetest places the flags on its own, so comparing with
// it checks where AddArtifact puts each one.
func dataWithArtifacts(flags ...int) []byte {
	return savetest.Data(&savetest.Slot{
		Town: "Tipa", Key: 29,
		Characters: []savetest.Character{{Name: "Alba", Gil: 10, Artifacts: flags}},
	})
}

// TestAddArtifactSetsTheGamesFlag grants every artifact in turn and checks
// that the result is byte for byte the save that owns that one flag.
func TestAddArtifactSetsTheGamesFlag(t *testing.T) {
	for id := save.ArtifactLimit.Min; id <= save.ArtifactLimit.Max; id++ {
		before := mustParse(t, dataWithArtifacts())
		after := mustParse(t, dataWithArtifacts())
		s := mustSlot(t, after, 0)
		c := mustCharacter(t, s, 0)

		if err := c.AddArtifact(id); err != nil {
			t.Fatalf("AddArtifact(%#x): %v", id, err)
		}
		if err := after.SetSlot(0, s); err != nil {
			t.Fatalf("SetSlot: %v", err)
		}

		flag := id - 0x9F
		if !bytes.Equal(after.Bytes(), dataWithArtifacts(flag)) {
			t.Fatalf("AddArtifact(%#x) did not produce the save with flag %d set", id, flag)
		}
		if err := save.VerifyEdit(before, after, 0, c.ArtifactRange(id)); err != nil {
			t.Errorf("VerifyEdit after AddArtifact(%#x): %v", id, err)
		}
	}
}

func TestAddArtifactKeepsOtherArtifactsAndFields(t *testing.T) {
	d := mustParse(t, dataWithArtifacts(1, 24, 60, 69))
	s := mustSlot(t, d, 0)
	c := mustCharacter(t, s, 0)

	if err := c.AddArtifact(0xE5); err != nil { // Moon Pendant, flag 70
		t.Fatalf("AddArtifact: %v", err)
	}
	if err := d.SetSlot(0, s); err != nil {
		t.Fatalf("SetSlot: %v", err)
	}

	reloaded := mustCharacter(t, mustSlot(t, mustParse(t, d.Bytes()), 0), 0)
	if got, want := reloaded.Artifacts(), []int{0xA0, 0xB7, 0xDB, 0xE4, 0xE5}; !slices.Equal(got, want) {
		t.Errorf("artifacts = %#x, want %#x", got, want)
	}
	if !reloaded.HasArtifact(0xE5) || reloaded.HasArtifact(0xE6) {
		t.Errorf("HasArtifact(0xe5) = %v, HasArtifact(0xe6) = %v, want true, false",
			reloaded.HasArtifact(0xE5), reloaded.HasArtifact(0xE6))
	}
	// The stored stats are the game's to recalculate; granting an
	// artifact must leave them alone.
	if reloaded.Gil() != 10 || reloaded.MaxHP() != 8 || reloaded.HP() != 8 || reloaded.Strength() != 10 {
		t.Errorf("other fields changed: gil %d, hp %d/%d, strength %d",
			reloaded.Gil(), reloaded.HP(), reloaded.MaxHP(), reloaded.Strength())
	}
}

func TestAddArtifactTwiceIsRefused(t *testing.T) {
	d := mustParse(t, dataWithArtifacts())
	s := mustSlot(t, d, 0)
	c := mustCharacter(t, s, 0)

	if err := c.AddArtifact(0xB7); err != nil {
		t.Fatalf("first AddArtifact: %v", err)
	}
	if err := c.AddArtifact(0xB7); !errors.Is(err, save.ErrAlreadyOwned) {
		t.Errorf("second AddArtifact = %v, want ErrAlreadyOwned", err)
	}
	if got, want := c.Artifacts(), []int{0xB7}; !slices.Equal(got, want) {
		t.Errorf("artifacts = %#x, want %#x", got, want)
	}
}

func TestAddArtifactRejects(t *testing.T) {
	s := mustSlot(t, mustParse(t, dataWithArtifacts()), 0)
	c := mustCharacter(t, s, 0)

	outside := []int{
		-1, 0,
		0x9E,  // Force Ring, the last accessory
		0xE8,  // first unnamed placeholder
		0xFE,  // last flag a character has
		0xFF,  // past the flags
		0x125, // Phoenix Down
	}
	for _, id := range outside {
		if err := c.AddArtifact(id); !errors.Is(err, save.ErrOutOfRange) {
			t.Errorf("AddArtifact(%#x) = %v, want ErrOutOfRange", id, err)
		}
	}
	if got := c.Artifacts(); len(got) != 0 {
		t.Errorf("artifacts = %#x after refused edits, want none", got)
	}

	if err := mustCharacter(t, s, 4).AddArtifact(0x9F); err == nil {
		t.Error("AddArtifact on an unused record succeeded, want an error")
	}
}
