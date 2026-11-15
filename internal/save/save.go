// Package save models the game's save data inside a GCI: the four save
// slots, their obfuscation, their checksum and the fields of each
// character record.
//
// Layout and algorithms follow the FFCC decompilation project
// (https://github.com/zcanann/FFCC-Decomp), mainly src/memorycard.cpp,
// include/ffcc/memorycard.h and src/wm_menu.cpp. Only the PAL version of
// the game is supported.
//
// Every setter in this package is limited to values the game itself can
// produce, and names the decomp code that shows it.
package save

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	// BannerSize is the size of the area in front of the slots. It holds
	// the comment, banner and icons shown by the memory card manager.
	BannerSize = 0x4000
	// SlotSize is the size of one save slot on the card.
	SlotSize = 0xA000
	// SlotCount is the number of save slots in the file. The game reads
	// and writes slot n at BannerSize + n*SlotSize (McCtrl in wm_menu.cpp).
	SlotCount = 4
	// DataSize is the size of the game's file, the value it passes to
	// CARDCreateAsync: 22 blocks.
	DataSize = BannerSize + SlotCount*SlotSize

	// PAL identity of the save file on the memory card.
	palGameCode  = "GCCP"
	palMakerCode = "01"
	palFileName  = "FFCC" // CardConst::MC_FNAME
	palRegion    = 'E'    // kMemoryCardRegion for GCCP01
)

// Slot header fields the game checks before it accepts a slot
// (CMemoryCardMan::SetLoadData and McCtrl::SetListDat).
const (
	offMaker   = 0x00 // "GDS"
	offTitle   = 0x04 // "FFCC"
	offMachine = 0x08 // "GC"
	offVersion = 0x0C // "1.00"
	offRegion  = 0x10

	offFrameCounter = 0x2C   // big-endian u32, play time in frames
	offPartySlots   = 0x30   // 4 big-endian s32 character indices, -1 for none
	offTownName     = 0x10C0 // 16 bytes, NUL padded
	offCharacters   = 0x14D0 // CharacterCount records of characterSize bytes

	townNameLen = 16
	partySize   = 4

	// framesPerMinute converts the frame counter to play time on PAL
	// (kMemoryCardFramesPerMinute).
	framesPerMinute = 1500
	maxPlayHours    = 999
)

var (
	// ErrUnsupported reports a save file this tool does not handle.
	ErrUnsupported = errors.New("unsupported save file")
	// ErrEmptySlot reports a slot that holds no save.
	ErrEmptySlot = errors.New("slot is empty")
	// ErrCorruptSlot reports a slot the game would not load.
	ErrCorruptSlot = errors.New("slot does not hold a valid save")
)

// CheckIdentity reports whether a GCI with the given header fields is a
// PAL Final Fantasy Crystal Chronicles save.
func CheckIdentity(gameCode, makerCode, fileName string) error {
	if gameCode == palGameCode && makerCode == palMakerCode && fileName == palFileName {
		return nil
	}
	switch gameCode {
	case "GCCE", "GCCJ":
		return fmt.Errorf("%w: game code %s is a non-PAL version of the game; only PAL (%s) is supported",
			ErrUnsupported, gameCode, palGameCode)
	}
	return fmt.Errorf("%w: got game %q maker %q file %q, want %q %q %q",
		ErrUnsupported, gameCode, makerCode, fileName, palGameCode, palMakerCode, palFileName)
}

// Data is the game's save file: the block data of the GCI.
type Data struct {
	b []byte
}

// Parse wraps the block data of a GCI. The returned Data does not alias b.
func Parse(b []byte) (*Data, error) {
	if len(b) != DataSize {
		return nil, fmt.Errorf("%w: save data is %d bytes, want %d", ErrUnsupported, len(b), DataSize)
	}
	return &Data{b: bytes.Clone(b)}, nil
}

// Bytes returns the save file as stored on the card.
func (d *Data) Bytes() []byte {
	return bytes.Clone(d.b)
}

func (d *Data) stored(index int) []byte {
	start := BannerSize + index*SlotSize
	return d.b[start : start+SlotSize]
}

func checkSlotIndex(index int) error {
	if index < 0 || index >= SlotCount {
		return fmt.Errorf("slot index %d is out of range 0-%d", index, SlotCount-1)
	}
	return nil
}

// Slot decodes and validates the slot with the given zero-based index.
// It returns ErrEmptySlot for a slot that was never written and
// ErrCorruptSlot for one that fails the checks the game applies on load.
func (d *Data) Slot(index int) (*Slot, error) {
	if err := checkSlotIndex(index); err != nil {
		return nil, err
	}

	stored := d.stored(index)
	if isZero(stored) {
		return nil, ErrEmptySlot
	}

	plain := Decode(stored)
	if err := validate(plain); err != nil {
		return nil, err
	}
	return &Slot{plain: plain}, nil
}

// SetSlot stores s in the slot with the given zero-based index. It
// recomputes the checksum and re-applies the obfuscation with the slot's
// existing key.
func (d *Data) SetSlot(index int, s *Slot) error {
	if err := checkSlotIndex(index); err != nil {
		return err
	}

	plain := bytes.Clone(s.plain)
	binary.BigEndian.PutUint32(plain[offCRC:], Checksum(plain))
	if err := validate(plain); err != nil {
		return fmt.Errorf("refusing to store slot: %w", err)
	}

	copy(d.stored(index), Encode(plain))
	return nil
}

// validate applies the checks the game makes before it lists and loads a
// slot: the four header tags, the region, the checksum and a non-empty
// town name.
func validate(plain []byte) error {
	tags := []struct {
		name string
		off  int
		want string
	}{
		{"maker", offMaker, "GDS"},
		{"title", offTitle, "FFCC"},
		{"machine", offMachine, "GC"},
		{"version", offVersion, "1.00"},
	}
	for _, tag := range tags {
		got := plain[tag.off : tag.off+len(tag.want)]
		if string(got) != tag.want {
			return fmt.Errorf("%w: %s tag is %q, want %q", ErrCorruptSlot, tag.name, got, tag.want)
		}
	}

	if region := plain[offRegion]; region != palRegion {
		return fmt.Errorf("%w: region is %q, want %q (PAL)", ErrCorruptSlot, region, palRegion)
	}

	stored := binary.BigEndian.Uint32(plain[offCRC:])
	if sum := Checksum(plain); sum != stored {
		return fmt.Errorf("%w: checksum is %08x, computed %08x", ErrCorruptSlot, stored, sum)
	}

	if plain[offTownName] == 0 {
		return fmt.Errorf("%w: town name is empty", ErrCorruptSlot)
	}
	return nil
}

func isZero(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}

// Slot is one decoded save slot.
type Slot struct {
	plain []byte
}

// TownName returns the name the player gave their home town.
func (s *Slot) TownName() string {
	return text(s.plain[offTownName : offTownName+townNameLen])
}

// PlayTime returns the play time the game shows for the slot
// (CMemoryCardMan::CnvPlayTime).
func (s *Slot) PlayTime() (hours, minutes int) {
	total := int(binary.BigEndian.Uint32(s.plain[offFrameCounter:]) / framesPerMinute)
	return min(total/60, maxPlayHours), total % 60
}

// InParty reports whether the character with the given zero-based index
// is in the active party.
func (s *Slot) InParty(character int) bool {
	for i := range partySize {
		if int(int32(binary.BigEndian.Uint32(s.plain[offPartySlots+4*i:]))) == character {
			return true
		}
	}
	return false
}

// Character returns the character record with the given zero-based index.
// Records exist for all CharacterCount indices; use Character.Exists to
// tell whether one holds a character.
func (s *Slot) Character(index int) (*Character, error) {
	if index < 0 || index >= CharacterCount {
		return nil, fmt.Errorf("character index %d is out of range 0-%d", index, CharacterCount-1)
	}
	start := offCharacters + index*characterSize
	return &Character{index: index, b: s.plain[start : start+characterSize]}, nil
}

// text converts a NUL-padded name to a string. The game uses its own
// character encoding outside ASCII, so any other byte is shown as "?".
func text(b []byte) string {
	if i := bytes.IndexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	out := make([]byte, len(b))
	for i, c := range b {
		if c < 0x20 || c > 0x7E {
			c = '?'
		}
		out[i] = c
	}
	return string(out)
}
