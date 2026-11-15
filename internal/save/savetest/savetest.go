// Package savetest builds synthetic PAL save files for tests.
//
// Real save files cannot live in the repository: they are personal data
// and embed the game's banner art. The files built here contain only what
// the editor reads, with every other byte zero.
//
// The field offsets are written out here on purpose instead of being
// imported from the save package, so that a wrong offset in either place
// makes a test fail.
package savetest

import (
	"encoding/binary"

	"github.com/ailequal/ffcc-gci/internal/save"
)

// Character describes one character record.
type Character struct {
	Name      string
	Tribe     int
	Gil       int
	Items     []int // item IDs, placed in inventory slots 0, 1, 2...
	Artifacts []int // artifact flag indices
}

// Slot describes one save slot.
type Slot struct {
	Town       string
	Frames     uint32 // play time in frames
	Key        byte   // obfuscation key
	Characters []Character
}

// GCI returns a complete GCI file holding the given slots in order. A nil
// entry leaves that slot empty, as do slots beyond the ones given.
func GCI(slots ...*Slot) []byte {
	header := make([]byte, 0x40)
	copy(header[0x00:], "GCCP")
	copy(header[0x04:], "01")
	header[0x06] = 0xFF
	copy(header[0x08:], "FFCC")
	binary.BigEndian.PutUint16(header[0x38:], 22)
	return append(header, Data(slots...)...)
}

// Data returns the block data of the file GCI builds.
func Data(slots ...*Slot) []byte {
	data := make([]byte, save.DataSize)
	copy(data, "FF Crystal Chronicles")
	for i, slot := range slots {
		if slot == nil {
			continue
		}
		start := save.BannerSize + i*save.SlotSize
		copy(data[start:], slot.stored())
	}
	return data
}

func (s *Slot) stored() []byte {
	p := make([]byte, save.SlotSize)
	be := binary.BigEndian

	copy(p[0x00:], "GDS")
	copy(p[0x04:], "FFCC")
	copy(p[0x08:], "GC")
	copy(p[0x0C:], "1.00")
	p[0x10] = 'E'
	p[0x11] = s.Key
	be.PutUint32(p[0x18:], 0x1234ABCD) // m_random
	be.PutUint32(p[0x2C:], s.Frames)
	for i := range 4 {
		member := int32(-1)
		if i < len(s.Characters) {
			member = int32(i)
		}
		be.PutUint32(p[0x30+4*i:], uint32(member))
	}
	copy(p[0x10C0:], s.Town)

	for i, c := range s.Characters {
		r := p[0x14D0+i*0x9C0:][:0x9C0]
		be.PutUint16(r[0x06:], 8) // max HP
		be.PutUint16(r[0x08:], 8) // HP
		be.PutUint16(r[0x0A:], 10)
		be.PutUint16(r[0x0C:], 11)
		be.PutUint16(r[0x0E:], 12)
		be.PutUint16(r[0x28:], uint16(len(c.Items)))
		be.PutUint16(r[0x2E:], uint16(c.Tribe))
		for e := range 4 {
			be.PutUint16(r[0x34+2*e:], 0xFFFF) // nothing equipped
		}
		for slot := range 64 {
			id := uint16(0xFFFF)
			if slot < len(c.Items) {
				id = uint16(c.Items[slot])
			}
			be.PutUint16(r[0x3C+2*slot:], id)
		}
		for _, a := range c.Artifacts {
			word := r[0xBC+4*(a/32):]
			be.PutUint32(word, be.Uint32(word)|1<<(a%32))
		}
		be.PutUint32(r[0xEC:], uint32(int32(c.Gil)))
		copy(r[0xF0:], c.Name)
		be.PutUint32(r[0x5B4:], 1) // exists
	}

	be.PutUint32(p[0x1C:], save.Checksum(p))
	return save.Encode(p)
}
