package itemdb

import (
	_ "embed"
	"fmt"
	"strconv"
	"strings"
)

// The game describes every item with one of a small set of texts, such as
// "Material for equipment" or "Design for Lilties", plus a number for
// bonuses like "Strength +2". Which text an item gets is fixed in the
// game's GBA client (gItemDescIds and gItemDescs_En in the decompilation's
// gba/src/cli/main/msg.c).
//
// That table is the per-item evidence for what an item is. Names alone
// are not: placeholders have names too.

//go:embed desc_texts_pal_en.txt
var descTextsFile string

//go:embed desc_ids_pal.txt
var descIDsFile string

// Indices into the description texts.
const (
	descUnknown      = 0  // "???"
	descPhoenixDown  = 34 // "Revives the fallen"
	descMaterial     = 35 // "Material for equipment"
	descFood         = 47 // "Restores HP"
	descDesignClavat = 50 // "Design for Clavats"
	descDesignLilty  = 51
	descDesignYuke   = 52
	descDesignSelkie = 53
	descDesignMen    = 54
	descDesignWomen  = 55
	descDesignAnyone = 56
)

var descTexts = strings.Split(strings.TrimSuffix(descTextsFile, "\n"), "\n")

// descIDs holds, for each item ID, the index of its description text and
// the number shown after it (0 for none).
var descIDs = func() [][2]int {
	lines := strings.Split(strings.TrimSuffix(descIDsFile, "\n"), "\n")
	out := make([][2]int, len(lines))
	for i, line := range lines {
		text, number, ok := strings.Cut(line, " ")
		a, errA := strconv.Atoi(text)
		b, errB := strconv.Atoi(number)
		if !ok || errA != nil || errB != nil || a < 0 || a >= len(descTexts) {
			panic(fmt.Sprintf("itemdb: bad description entry %q for item %d", line, i))
		}
		out[i] = [2]int{a, b}
	}
	return out
}()

func descOf(id int) int {
	if id < 0 || id >= len(descIDs) {
		return descUnknown
	}
	return descIDs[id][0]
}

// Description returns the game's description of an item, for example
// "Material for equipment" or "Strength +2". The second result is false
// when the game has no description for the ID, which marks placeholders
// and unused entries.
func Description(id int) (string, bool) {
	d := descOf(id)
	if d == descUnknown {
		return "", false
	}
	text := descTexts[d]
	if n := descIDs[id][1]; n > 0 {
		text = fmt.Sprintf("%s +%d", text, n)
	}
	return text, true
}

// Kind is what the game's description says an item is.
type Kind int

// Kinds of item that matter to the editor. Every other item is KindOther.
const (
	KindOther Kind = iota
	KindPhoenixDown
	KindMaterial
	KindFood
	KindDesign
)

func (k Kind) String() string {
	switch k {
	case KindPhoenixDown:
		return "Phoenix Down"
	case KindMaterial:
		return "Materials"
	case KindFood:
		return "Food"
	case KindDesign:
		return "Designs"
	}
	return "Other"
}

// KindOf returns the kind of the item with the given ID.
func KindOf(id int) Kind {
	switch d := descOf(id); {
	case d == descPhoenixDown:
		return KindPhoenixDown
	case d == descMaterial:
		return KindMaterial
	case d == descFood:
		return KindFood
	case d >= descDesignClavat && d <= descDesignAnyone:
		return KindDesign
	}
	return KindOther
}

// DesignUser is who can forge the equipment of a design.
type DesignUser int

// The users a design can be for. The four tribes have the values of the
// game's tribe IDs.
const (
	ForClavats DesignUser = iota
	ForLilties
	ForYukes
	ForSelkies
	ForMen
	ForWomen
	ForAnyone
)

// DesignFor returns who a design is for. The second result is false when
// the item is not a design.
func DesignFor(id int) (DesignUser, bool) {
	d := descOf(id)
	if d < descDesignClavat || d > descDesignAnyone {
		return 0, false
	}
	return DesignUser(d - descDesignClavat), true
}
