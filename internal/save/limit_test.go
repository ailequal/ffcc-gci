package save_test

import (
	"errors"
	"testing"

	"github.com/ailequal/ffcc-gci/internal/save"
)

func TestLimitsAreWellFormed(t *testing.T) {
	limits := save.Limits()
	if len(limits) == 0 {
		t.Fatal("Limits() is empty")
	}

	seen := make(map[string]bool)
	for _, l := range limits {
		if l.Name == "" || l.Source == "" {
			t.Errorf("limit %+v lacks a name or a source", l)
		}
		if l.Min > l.Max {
			t.Errorf("limit %q has Min %d above Max %d", l.Name, l.Min, l.Max)
		}
		if seen[l.Name] {
			t.Errorf("limit %q is listed twice", l.Name)
		}
		seen[l.Name] = true
	}
}

func TestLimitCheck(t *testing.T) {
	l := save.Limit{Name: "example", Min: 3, Max: 7, Source: "test"}

	for _, v := range []int{3, 5, 7} {
		if err := l.Check(v); err != nil {
			t.Errorf("Check(%d) = %v, want nil", v, err)
		}
	}
	for _, v := range []int{-1, 2, 8, 1 << 40} {
		if err := l.Check(v); !errors.Is(err, save.ErrOutOfRange) {
			t.Errorf("Check(%d) = %v, want ErrOutOfRange", v, err)
		}
	}
}

// TestSetGilEnforcesLimit checks the exact edges of the gil range: both
// ends are accepted and the first value past each end is refused.
func TestSetGilEnforcesLimit(t *testing.T) {
	limit := save.GilLimit
	if limit.Min != 0 || limit.Max != 99_999_999 {
		t.Fatalf("GilLimit = %d-%d, want 0-99999999 as in CCaravanWork::AddGil", limit.Min, limit.Max)
	}

	s := mustSlot(t, mustParse(t, sample()), 0)
	c := mustCharacter(t, s, 0)

	for _, gil := range []int{limit.Min, limit.Max} {
		if err := c.SetGil(gil); err != nil {
			t.Errorf("SetGil(%d) = %v, want nil", gil, err)
		}
		if c.Gil() != gil {
			t.Errorf("gil = %d after SetGil(%d)", c.Gil(), gil)
		}
	}
	for _, gil := range []int{limit.Min - 1, limit.Max + 1} {
		if err := c.SetGil(gil); !errors.Is(err, save.ErrOutOfRange) {
			t.Errorf("SetGil(%d) = %v, want ErrOutOfRange", gil, err)
		}
	}
	if c.Gil() != limit.Max {
		t.Errorf("gil = %d after refused edits, want it unchanged at %d", c.Gil(), limit.Max)
	}
}

func TestArtifactLimitCoversTheNamedArtifacts(t *testing.T) {
	limit := save.ArtifactLimit
	if limit.Min != 0x9F || limit.Max != 0xE7 {
		t.Fatalf("ArtifactLimit = %#x-%#x, want 0x9f-0xe7 (Shuriken to Sun Pendant)", limit.Min, limit.Max)
	}
}
