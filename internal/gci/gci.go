// Package gci reads and writes the GCI container: the 0x40-byte GameCube
// directory entry that Dolphin stores in front of a save file's raw
// blocks.
//
// The package only knows about the container. It checks that the file is
// structurally a GCI and exposes the header fields and the block data; it
// does not interpret the game's own data. The header layout follows the
// DEntry struct in Dolphin's Source/Core/Core/HW/GCMemcard/GCMemcard.h.
package gci

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	// HeaderSize is the size of the directory entry at the start of a GCI.
	HeaderSize = 0x40
	// BlockSize is the size of one memory card block.
	BlockSize = 0x2000

	// Offsets inside the header.
	offGameCode   = 0x00 // 4 bytes
	offMakerCode  = 0x04 // 2 bytes
	offFileName   = 0x08 // 0x20 bytes, NUL padded
	offBlockCount = 0x38 // big-endian u16

	fileNameLen = 0x20
)

// ErrInvalid reports a file that is not a well-formed GCI.
var ErrInvalid = errors.New("not a valid GCI file")

// File is a parsed GCI: the directory entry and the save's block data.
type File struct {
	// Header is the directory entry, kept byte for byte.
	Header [HeaderSize]byte
	// Data is the block data that follows the header. Its length is
	// always BlockCount() * BlockSize.
	Data []byte
}

// Parse splits raw GCI bytes into header and block data. It applies the
// same size rule Dolphin uses when it loads a GCI: the file must be
// exactly the header plus the number of blocks the header declares.
// The returned File does not alias b.
func Parse(b []byte) (*File, error) {
	if len(b) < HeaderSize {
		return nil, fmt.Errorf("%w: %d bytes is smaller than the %d-byte header", ErrInvalid, len(b), HeaderSize)
	}

	f := &File{}
	copy(f.Header[:], b)

	want := HeaderSize + f.BlockCount()*BlockSize
	if f.BlockCount() == 0 || len(b) != want {
		return nil, fmt.Errorf("%w: header declares %d blocks (%d bytes), file has %d bytes",
			ErrInvalid, f.BlockCount(), want, len(b))
	}

	f.Data = bytes.Clone(b[HeaderSize:])
	return f, nil
}

// Bytes returns the GCI as it is stored on disk.
func (f *File) Bytes() []byte {
	out := make([]byte, 0, HeaderSize+len(f.Data))
	out = append(out, f.Header[:]...)
	return append(out, f.Data...)
}

// GameCode returns the four-character game code, for example "GCCP".
func (f *File) GameCode() string {
	return string(f.Header[offGameCode : offGameCode+4])
}

// MakerCode returns the two-character maker code, for example "01".
func (f *File) MakerCode() string {
	return string(f.Header[offMakerCode : offMakerCode+2])
}

// FileName returns the save's internal file name without NUL padding.
func (f *File) FileName() string {
	name := f.Header[offFileName : offFileName+fileNameLen]
	if i := bytes.IndexByte(name, 0); i >= 0 {
		name = name[:i]
	}
	return string(name)
}

// BlockCount returns the number of blocks the header declares.
func (f *File) BlockCount() int {
	return int(binary.BigEndian.Uint16(f.Header[offBlockCount:]))
}
