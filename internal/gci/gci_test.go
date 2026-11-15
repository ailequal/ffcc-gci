package gci

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
)

// build returns GCI bytes whose header declares declared blocks and whose
// data holds actual blocks.
func build(declared, actual int) []byte {
	b := make([]byte, HeaderSize+actual*BlockSize)
	copy(b[offGameCode:], "GCCP")
	copy(b[offMakerCode:], "01")
	copy(b[offFileName:], "FFCC")
	binary.BigEndian.PutUint16(b[offBlockCount:], uint16(declared))
	for i := HeaderSize; i < len(b); i++ {
		b[i] = byte(i)
	}
	return b
}

func TestParse(t *testing.T) {
	raw := build(3, 3)

	f, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if got := f.GameCode(); got != "GCCP" {
		t.Errorf("GameCode = %q, want %q", got, "GCCP")
	}
	if got := f.MakerCode(); got != "01" {
		t.Errorf("MakerCode = %q, want %q", got, "01")
	}
	if got := f.FileName(); got != "FFCC" {
		t.Errorf("FileName = %q, want %q", got, "FFCC")
	}
	if got := f.BlockCount(); got != 3 {
		t.Errorf("BlockCount = %d, want 3", got)
	}
	if got := len(f.Data); got != 3*BlockSize {
		t.Errorf("len(Data) = %d, want %d", got, 3*BlockSize)
	}
	if !bytes.Equal(f.Bytes(), raw) {
		t.Error("Bytes() does not reproduce the input")
	}
}

func TestParseDoesNotAliasInput(t *testing.T) {
	raw := build(1, 1)
	f, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	want := f.Data[0]
	raw[HeaderSize] ^= 0xFF

	if f.Data[0] != want {
		t.Error("changing the input changed the parsed file")
	}
}

func TestParseRejects(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
	}{
		{name: "empty", raw: nil},
		{name: "shorter than header", raw: make([]byte, HeaderSize-1)},
		{name: "zero blocks", raw: build(0, 0)},
		{name: "fewer blocks than declared", raw: build(3, 2)},
		{name: "more blocks than declared", raw: build(3, 4)},
		{name: "trailing byte", raw: append(build(2, 2), 0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Parse(tt.raw); !errors.Is(err, ErrInvalid) {
				t.Errorf("Parse error = %v, want ErrInvalid", err)
			}
		})
	}
}
