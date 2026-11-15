package save

import (
	"encoding/binary"
	"fmt"
)

const (
	// CharacterCount is the number of character records in a slot.
	CharacterCount = 8
	// InventorySize is the number of inventory slots a character has.
	InventorySize = 64
	// EmptyItem marks an unused inventory slot.
	EmptyItem = -1
	// MaxGil is the most gil a character can hold. The game clamps to
	// this value in CCaravanWork::AddGil and refuses to exceed it in
	// CCaravanWork::CanAddGil (src/gobjwork.cpp).
	MaxGil = 99_999_999

	characterSize = 0x9C0
	nameLen       = 16
	artifactWords = 3
)

// Offsets inside a character record (Mc::CharaDat in memorycard.h). All
// multi-byte fields are big-endian.
const (
	offMaxHP         = 0x06  // u16, in half hearts
	offHP            = 0x08  // u16, in half hearts
	offStrength      = 0x0A  // u16
	offMagic         = 0x0C  // u16
	offDefense       = 0x0E  // u16
	offItemCount     = 0x28  // u16
	offTribe         = 0x2E  // u16
	offInventory     = 0x3C  // InventorySize s16 item IDs
	offArtifactFlags = 0xBC  // artifactWords u32 bit sets
	offGil           = 0xEC  // s32
	offName          = 0xF0  // nameLen bytes, NUL padded
	offExists        = 0x5B4 // s32, non-zero when the record is in use
)

// Tribe is a character's tribe.
type Tribe int

// Tribe IDs, in the order of the game's tribe name table.
const (
	Clavat Tribe = iota
	Lilty
	Yuke
	Selkie
)

func (t Tribe) String() string {
	switch t {
	case Clavat:
		return "Clavat"
	case Lilty:
		return "Lilty"
	case Yuke:
		return "Yuke"
	case Selkie:
		return "Selkie"
	}
	return fmt.Sprintf("tribe %d", int(t))
}

// Range is a half-open range of byte offsets inside a decoded slot.
type Range struct {
	Start, End int
}

// Character is a view of one character record inside a Slot. Setters
// change the slot the character came from.
type Character struct {
	index int
	b     []byte
}

func (c *Character) u16(off int) int { return int(binary.BigEndian.Uint16(c.b[off:])) }
func (c *Character) s16(off int) int { return int(int16(binary.BigEndian.Uint16(c.b[off:]))) }
func (c *Character) s32(off int) int { return int(int32(binary.BigEndian.Uint32(c.b[off:]))) }

// Index returns the zero-based index of the record in its slot.
func (c *Character) Index() int { return c.index }

// Exists reports whether the record holds a character.
func (c *Character) Exists() bool { return c.s32(offExists) != 0 }

// Name returns the character's name.
func (c *Character) Name() string { return text(c.b[offName : offName+nameLen]) }

// Tribe returns the character's tribe.
func (c *Character) Tribe() Tribe { return Tribe(c.u16(offTribe)) }

// HP returns the current hit points, in half hearts.
func (c *Character) HP() int { return c.u16(offHP) }

// MaxHP returns the maximum hit points, in half hearts.
func (c *Character) MaxHP() int { return c.u16(offMaxHP) }

// Strength returns the strength value as of the last save.
func (c *Character) Strength() int { return c.u16(offStrength) }

// Magic returns the magic value as of the last save.
func (c *Character) Magic() int { return c.u16(offMagic) }

// Defense returns the defense value as of the last save.
func (c *Character) Defense() int { return c.u16(offDefense) }

// Gil returns the character's money.
func (c *Character) Gil() int { return c.s32(offGil) }

// ItemCount returns the number of used inventory slots as the save
// records it.
func (c *Character) ItemCount() int { return c.u16(offItemCount) }

// Inventory returns the item ID in each inventory slot, EmptyItem for an
// unused one.
func (c *Character) Inventory() [InventorySize]int {
	var items [InventorySize]int
	for i := range items {
		items[i] = c.s16(offInventory + 2*i)
	}
	return items
}

// Artifacts returns the indices of the artifact flags that are set, in
// ascending order. Flag n stands for item itemdb.ArtifactBase+n.
func (c *Character) Artifacts() []int {
	var set []int
	for word := range artifactWords {
		flags := binary.BigEndian.Uint32(c.b[offArtifactFlags+4*word:])
		for bit := range 32 {
			if flags&(1<<bit) != 0 {
				set = append(set, word*32+bit)
			}
		}
	}
	return set
}

// GilRange returns where the character's gil is stored in the decoded
// slot.
func (c *Character) GilRange() Range {
	start := offCharacters + c.index*characterSize + offGil
	return Range{Start: start, End: start + 4}
}

// SetGil sets the character's money. The value must be within GilLimit,
// the range the game itself keeps gil in.
func (c *Character) SetGil(gil int) error {
	if !c.Exists() {
		return fmt.Errorf("character record %d is not in use", c.index)
	}
	if err := GilLimit.Check(gil); err != nil {
		return err
	}
	binary.BigEndian.PutUint32(c.b[offGil:], uint32(gil))
	return nil
}
