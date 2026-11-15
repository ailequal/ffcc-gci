package save

import (
	"bytes"
	"math/rand/v2"
	"testing"
)

func TestCRCTableMatchesGame(t *testing.T) {
	// Entries copied from s_CrcTable in the decomp's src/memorycard.cpp.
	want := map[int]uint32{
		0:   0x00000000,
		1:   0x04c11db7,
		2:   0x09823b6e,
		8:   0x2608edb8,
		252: 0xbcb4666d,
		255: 0xb1f740b4,
	}
	for i, w := range want {
		if crcTable[i] != w {
			t.Errorf("crcTable[%d] = %08x, want %08x", i, crcTable[i], w)
		}
	}
}

func TestCRCUpdateKnownAnswer(t *testing.T) {
	// The standard check value of CRC-32/MPEG-2, which has the same
	// polynomial, bit order and initial value as the game's routine.
	if got, want := crcUpdate(0xFFFFFFFF, []byte("123456789")), uint32(0x0376E6E7); got != want {
		t.Errorf("crcUpdate = %08x, want %08x", got, want)
	}
}

func TestChecksumCoverage(t *testing.T) {
	plain := make([]byte, SlotSize)
	base := Checksum(plain)

	covered := []int{0x00, offRotateKey, offObfuscated, offCRC - 1, offCRC + 4, saveDatSize - 1}
	for _, off := range covered {
		plain[off] ^= 0xFF
		if Checksum(plain) == base {
			t.Errorf("changing byte %#x did not change the checksum", off)
		}
		plain[off] ^= 0xFF
	}

	ignored := []int{offCRC, offCRC + 3, saveDatSize, SlotSize - 1}
	for _, off := range ignored {
		plain[off] ^= 0xFF
		if Checksum(plain) != base {
			t.Errorf("changing byte %#x changed the checksum", off)
		}
		plain[off] ^= 0xFF
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))

	for key := range 256 {
		stored := make([]byte, SlotSize)
		for i := range stored {
			stored[i] = byte(rng.Uint32())
		}
		stored[offRotateKey] = byte(key)

		plain := Decode(stored)
		if !bytes.Equal(plain[:offObfuscated], stored[:offObfuscated]) {
			t.Fatalf("key %d: Decode changed the plain header", key)
		}
		if !bytes.Equal(Encode(plain), stored) {
			t.Fatalf("key %d: Encode(Decode(x)) != x", key)
		}
	}
}

func TestDecodeKnownWord(t *testing.T) {
	// With key 4, the game word 0x12345678 is rotated left to 0x23456781
	// and stored little-endian.
	stored := make([]byte, SlotSize)
	stored[offRotateKey] = 4 + 32 // only the key modulo 32 matters
	copy(stored[offObfuscated:], []byte{0x81, 0x67, 0x45, 0x23})

	plain := Decode(stored)

	if got, want := plain[offObfuscated:offObfuscated+4], []byte{0x12, 0x34, 0x56, 0x78}; !bytes.Equal(got, want) {
		t.Errorf("decoded word = % x, want % x", got, want)
	}
}
