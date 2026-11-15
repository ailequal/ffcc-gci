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

// artifactGroups are the groups the game's item table sorts artifacts
// into, each with the item ID it starts at. CCaravanWork::CalcArtifactStatus
// adds an artifact's value to the stat of its group.
var artifactGroups = []struct {
	first  int
	effect string
}{
	{0x9F, "Strength"},
	{0xB6, "Magic"},
	{0xCC, "Defence"},
	{0xDB, "Command slots"},
	{0xDF, "Spell rings"},
	{0xE4, "Hearts"},
	{0xE8, ""}, // unnamed placeholders start here
}

// ArtifactEffect returns what the artifact with the given item ID
// improves, such as "Strength". The second result is false when the ID
// is not an artifact.
func ArtifactEffect(id int) (string, bool) {
	if id < artifactGroups[0].first {
		return "", false
	}
	effect := ""
	for _, g := range artifactGroups {
		if id < g.first {
			break
		}
		effect = g.effect
	}
	return effect, effect != ""
}
