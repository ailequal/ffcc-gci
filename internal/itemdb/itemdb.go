// Package itemdb is the table of item IDs and their names.
//
// The names are the PAL English strings, in item ID order, taken from the
// FFCC decompilation (gba/src/cli/main/msg_item.c). They are used for
// display only.
package itemdb

import (
	_ "embed"
	"strings"
)

const (
	// ArtifactBase is the item ID of the first artifact. A character's
	// artifact flag n stands for item ArtifactBase+n (see SetLoadData in
	// the decomp's src/memorycard.cpp).
	ArtifactBase = 0x9F
	// ArtifactCount is the number of artifact flags a character has.
	ArtifactCount = 96
)

//go:embed names_pal_en.txt
var namesFile string

var names = strings.Split(strings.TrimSuffix(namesFile, "\n"), "\n")

// Count returns the number of item IDs in the table.
func Count() int {
	return len(names)
}

// Name returns the name of the item with the given ID. The second result
// is false when the ID is outside the table.
func Name(id int) (string, bool) {
	if id < 0 || id >= len(names) {
		return "", false
	}
	return names[id], true
}
