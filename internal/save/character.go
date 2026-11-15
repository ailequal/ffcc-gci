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

	// FirstArtifact and LastArtifact are the item IDs of the first and
	// last artifact a character can be granted. A character has
	// artifactFlags flags, for the item IDs from FirstArtifact on
	// (CMemoryCardMan::SetLoadData), but the IDs after LastArtifact are
	// unnamed placeholders in the game's item table.
	FirstArtifact = 0x9F
	LastArtifact  = 0xE7

	characterSize = 0x9C0
	nameLen       = 16
	artifactWords = 3
	artifactFlags = artifactWords * 32
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

// artifactBit locates the flag of an artifact: the offset of the byte
// that holds it inside the character record, and the bit in that byte.
// Flag n is bit n%32 of the big-endian 32-bit word n/32, as the game
// reads it in CMemoryCardMan::SetLoadData.
func artifactBit(id int) (off int, mask byte) {
	flag := id - FirstArtifact
	bit := flag % 32
	return offArtifactFlags + 4*(flag/32) + 3 - bit/8, 1 << (bit % 8)
}

// Artifacts returns the item IDs of the artifacts the character owns, in
// ascending order.
func (c *Character) Artifacts() []int {
	var owned []int
	for id := FirstArtifact; id < FirstArtifact+artifactFlags; id++ {
		if off, mask := artifactBit(id); c.b[off]&mask != 0 {
			owned = append(owned, id)
		}
	}
	return owned
}

// HasArtifact reports whether the character owns the artifact with the
// given item ID.
func (c *Character) HasArtifact(id int) bool {
	if id < FirstArtifact || id >= FirstArtifact+artifactFlags {
		return false
	}
	off, mask := artifactBit(id)
	return c.b[off]&mask != 0
}

// ArtifactRange returns where the flag of an artifact is stored in the
// decoded slot: the single byte that holds it.
func (c *Character) ArtifactRange(id int) Range {
	off, _ := artifactBit(id)
	start := offCharacters + c.index*characterSize + off
	return Range{Start: start, End: start + 1}
}

// AddArtifact grants the character an artifact, given by item ID within
// ArtifactLimit. It sets the artifact's flag, which is all the game does
// when a character receives one (CCaravanWork::SetArtifact).
//
// An artifact is a single flag, so it cannot be held twice: granting one
// the character already owns returns ErrAlreadyOwned and changes nothing.
//
// The strength, magic, defense and hit points stored in the save are not
// touched. They are the game's last calculation, and the game works them
// out again from the artifacts: hit points when it loads the save
// (SetLoadData), the others on every frame a character is on screen
// (CGPartyObj::onFrameAlways calling CCaravanWork::CalcStatus).
func (c *Character) AddArtifact(id int) error {
	if !c.Exists() {
		return fmt.Errorf("character record %d is not in use", c.index)
	}
	if err := ArtifactLimit.Check(id); err != nil {
		return err
	}
	if c.HasArtifact(id) {
		return ErrAlreadyOwned
	}
	off, mask := artifactBit(id)
	c.b[off] |= mask
	return nil
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
