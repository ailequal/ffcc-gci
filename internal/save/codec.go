package save

import (
	"encoding/binary"
	"fmt"
	"math/bits"
)

// Slot header offsets. The first offObfuscated bytes of a slot are stored
// as they are; everything after is obfuscated.
const (
	offRotateKey  = 0x11 // u8, rotation amount is this value modulo 32
	offObfuscated = 0x18 // first obfuscated byte
	offCRC        = 0x1C // big-endian u32

	// saveDatSize is the size of the game's save structure. The rest of
	// the slot is padding, which is obfuscated but not checksummed.
	saveDatSize = 0x8BD0
)

// crcTable is the table the game uses: CRC-32 with polynomial 0x04C11DB7,
// processed most significant bit first (s_CrcTable in memorycard.cpp).
var crcTable = func() (t [256]uint32) {
	for i := range t {
		c := uint32(i) << 24
		for range 8 {
			if c&0x80000000 != 0 {
				c = c<<1 ^ 0x04C11DB7
			} else {
				c <<= 1
			}
		}
		t[i] = c
	}
	return t
}()

func crcUpdate(crc uint32, p []byte) uint32 {
	for _, b := range p {
		crc = crc<<8 ^ crcTable[byte(crc>>24)^b]
	}
	return crc
}

// Checksum computes the checksum the game stores in a slot, from the
// slot's decoded bytes. It covers the save structure except the checksum
// field itself (CMemoryCardMan::CalcCrc).
func Checksum(plain []byte) uint32 {
	mustBeSlot(plain)
	crc := crcUpdate(0xFFFFFFFF, plain[:offCRC])
	crc = crcUpdate(crc, plain[offCRC+4:saveDatSize])
	return ^crc
}

// Decode returns the decoded bytes of a slot as stored on the card
// (CMemoryCardMan::DecodeData). Each obfuscated 32-bit word is stored
// little-endian and rotated left by the slot's key; decoding rotates it
// back and yields the big-endian layout the game works with.
func Decode(stored []byte) []byte {
	mustBeSlot(stored)
	plain := make([]byte, SlotSize)
	copy(plain, stored[:offObfuscated])

	key := int(stored[offRotateKey] % 32)
	for o := offObfuscated; o < SlotSize; o += 4 {
		w := binary.LittleEndian.Uint32(stored[o:])
		binary.BigEndian.PutUint32(plain[o:], bits.RotateLeft32(w, -key))
	}
	return plain
}

// Encode is the inverse of Decode (CMemoryCardMan::EncodeData). It uses
// the key already present in the slot header and does not touch the
// checksum.
func Encode(plain []byte) []byte {
	mustBeSlot(plain)
	stored := make([]byte, SlotSize)
	copy(stored, plain[:offObfuscated])

	key := int(plain[offRotateKey] % 32)
	for o := offObfuscated; o < SlotSize; o += 4 {
		w := binary.BigEndian.Uint32(plain[o:])
		binary.LittleEndian.PutUint32(stored[o:], bits.RotateLeft32(w, key))
	}
	return stored
}

func mustBeSlot(b []byte) {
	if len(b) != SlotSize {
		panic(fmt.Sprintf("save: slot buffer is %d bytes, want %d", len(b), SlotSize))
	}
}
