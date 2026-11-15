package itemdb

import "testing"

func TestName(t *testing.T) {
	tests := []struct {
		id     int
		want   string
		wantOK bool
	}{
		{id: 0x001, want: "Copper Sword", wantOK: true},
		{id: 0x034, want: "Aura Racket", wantOK: true},
		{id: 0x09F, want: "Shuriken", wantOK: true},
		{id: 0x0AA, want: "Mjöllnir", wantOK: true},
		{id: 0x125, want: "Phoenix Down", wantOK: true},
		{id: 0x17D, want: "Striped Apple", wantOK: true},
		{id: -1, wantOK: false},
		{id: Count(), wantOK: false},
	}

	for _, tt := range tests {
		got, ok := Name(tt.id)
		if got != tt.want || ok != tt.wantOK {
			t.Errorf("Name(%#x) = %q, %v; want %q, %v", tt.id, got, ok, tt.want, tt.wantOK)
		}
	}
}

func TestTableShape(t *testing.T) {
	if got, want := Count(), 501; got != want {
		t.Fatalf("Count() = %d, want %d", got, want)
	}
	for id := 0; id < Count(); id++ {
		if name, _ := Name(id); name == "" {
			t.Errorf("item %#x has an empty name", id)
		}
	}
}

func TestArtifactEffect(t *testing.T) {
	tests := []struct {
		id     int
		want   string
		wantOK bool
	}{
		{id: 0x9E, wantOK: false}, // Force Ring, an accessory
		{id: 0x9F, want: "Strength", wantOK: true},
		{id: 0xB5, want: "Strength", wantOK: true},
		{id: 0xB6, want: "Magic", wantOK: true},
		{id: 0xCB, want: "Magic", wantOK: true},
		{id: 0xCC, want: "Defence", wantOK: true},
		{id: 0xDA, want: "Defence", wantOK: true},
		{id: 0xDB, want: "Command slots", wantOK: true},
		{id: 0xDE, want: "Command slots", wantOK: true},
		{id: 0xDF, want: "Spell rings", wantOK: true},
		{id: 0xE3, want: "Spell rings", wantOK: true},
		{id: 0xE4, want: "Hearts", wantOK: true},
		{id: 0xE7, want: "Hearts", wantOK: true},
		{id: 0xE8, wantOK: false}, // placeholder
		{id: 0x125, wantOK: false},
	}

	for _, tt := range tests {
		got, ok := ArtifactEffect(tt.id)
		if got != tt.want || ok != tt.wantOK {
			t.Errorf("ArtifactEffect(%#x) = %q, %v; want %q, %v", tt.id, got, ok, tt.want, tt.wantOK)
		}
	}
}

func TestDescription(t *testing.T) {
	tests := []struct {
		id     int
		want   string
		wantOK bool
	}{
		{id: 0x09F, want: "Strength +1", wantOK: true},               // Shuriken
		{id: 0x0DB, want: "Gain another command slot", wantOK: true}, // Moogle Pocket
		{id: 0x0E7, want: "Gain another heart", wantOK: true},        // Sun Pendant
		{id: 0x0E8, wantOK: false},                                   // placeholder artifact flag
		{id: 293, want: "Revives the fallen", wantOK: true},
		{id: 294, want: "Material for equipment", wantOK: true},
		{id: 352, want: "Figurine of a goddess", wantOK: true},
		{id: 354, wantOK: false}, // Extra 22
		{id: 381, want: "Restores HP", wantOK: true},
		{id: 401, want: "Design for anyone", wantOK: true},
		{id: 411, want: "Design for Selkies", wantOK: true},
		{id: 494, wantOK: false}, // design 94
		{id: -1, wantOK: false},
		{id: 501, wantOK: false},
	}

	for _, tt := range tests {
		got, ok := Description(tt.id)
		if got != tt.want || ok != tt.wantOK {
			t.Errorf("Description(%d) = %q, %v; want %q, %v", tt.id, got, ok, tt.want, tt.wantOK)
		}
	}
}

func TestKindOf(t *testing.T) {
	tests := []struct {
		id   int
		want Kind
	}{
		{id: 1, want: KindOther},   // Copper Sword
		{id: 256, want: KindOther}, // Fire magicite
		{id: 293, want: KindPhoenixDown},
		{id: 294, want: KindMaterial},
		{id: 351, want: KindMaterial},
		{id: 352, want: KindOther}, // Goddess Stat.
		{id: 353, want: KindOther}, // Devil's Mask
		{id: 357, want: KindOther}, // Fruit Seed
		{id: 364, want: KindOther}, // Worn Bandana
		{id: 369, want: KindMaterial},
		{id: 380, want: KindOther}, // Extra 33
		{id: 381, want: KindFood},
		{id: 392, want: KindFood},
		{id: 393, want: KindOther}, // Flame Shard
		{id: 400, want: KindOther}, // Item Gil
		{id: 401, want: KindDesign},
		{id: 493, want: KindDesign},
		{id: 494, want: KindOther}, // design 94
	}

	for _, tt := range tests {
		if got := KindOf(tt.id); got != tt.want {
			t.Errorf("KindOf(%d) = %v, want %v", tt.id, got, tt.want)
		}
	}
}

func TestDesignFor(t *testing.T) {
	tests := []struct {
		id     int
		want   DesignUser
		wantOK bool
	}{
		{id: 401, want: ForAnyone, wantOK: true},  // Novice's Wpn.
		{id: 410, want: ForClavats, wantOK: true}, // Dark Wpn.
		{id: 408, want: ForLilties, wantOK: true}, // Hero's Wpn.
		{id: 491, want: ForYukes, wantOK: true},   // Forbid. Tome
		{id: 411, want: ForSelkies, wantOK: true}, // Lunar Wpn.
		{id: 475, want: ForMen, wantOK: true},
		{id: 471, want: ForWomen, wantOK: true},
		{id: 294, wantOK: false},
		{id: 494, wantOK: false},
	}

	for _, tt := range tests {
		got, ok := DesignFor(tt.id)
		if got != tt.want || ok != tt.wantOK {
			t.Errorf("DesignFor(%d) = %v, %v; want %v, %v", tt.id, got, ok, tt.want, tt.wantOK)
		}
	}
}

// TestArtifactEffectsMatchDescriptions cross-checks the artifact groups
// against the game's descriptions.
func TestArtifactEffectsMatchDescriptions(t *testing.T) {
	prefix := map[string]string{
		"Strength":      "Strength +",
		"Magic":         "Magic +",
		"Defence":       "Defence +",
		"Command slots": "Gain another command slot",
		"Spell rings":   "Cast ",
		"Hearts":        "Gain another heart",
	}
	for id := 0x9F; id <= 0xE7; id++ {
		effect, _ := ArtifactEffect(id)
		desc, ok := Description(id)
		if !ok || len(desc) < len(prefix[effect]) || desc[:len(prefix[effect])] != prefix[effect] {
			t.Errorf("artifact %d: effect %q, description %q", id, effect, desc)
		}
	}
}
