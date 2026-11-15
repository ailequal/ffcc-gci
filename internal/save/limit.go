package save

import (
	"errors"
	"fmt"
)

var (
	// ErrOutOfRange reports a value outside the range an editable field
	// accepts.
	ErrOutOfRange = errors.New("value out of range")
	// ErrAlreadyOwned reports an artifact the character already has.
	ErrAlreadyOwned = errors.New("the character already owns this artifact")
)

// Limit is the range of values an editable field accepts.
//
// Every value this package can change has a Limit, and its setter refuses
// anything outside it. A range is never a guess: Source names the code in
// the decompilation that shows the game keeps the value inside it.
type Limit struct {
	Name   string // what the value is, as shown to the user
	Min    int    // lowest accepted value
	Max    int    // highest accepted value
	Source string // why the range is safe
}

// Check returns an error wrapping ErrOutOfRange when v is outside the
// limit. The message states the accepted range.
func (l Limit) Check(v int) error {
	if v < l.Min || v > l.Max {
		return fmt.Errorf("%w: %s must be between %d and %d, got %d", ErrOutOfRange, l.Name, l.Min, l.Max, v)
	}
	return nil
}

// GilLimit is the range of a character's gil.
var GilLimit = Limit{
	Name: "gil",
	Min:  0,
	Max:  MaxGil,
	Source: "The game never lets gil leave this range: CCaravanWork::AddGil clamps to it " +
		"and CCaravanWork::CanAddGil refuses to go past it (src/gobjwork.cpp).",
}

// ArtifactLimit is the range of item IDs that can be granted as an
// artifact.
var ArtifactLimit = Limit{
	Name: "artifact item ID",
	Min:  FirstArtifact,
	Max:  LastArtifact,
	Source: "These are the game's 73 artifacts. A character has a flag for each, and for 23 more " +
		"item IDs that are unnamed placeholders and are refused (CMemoryCardMan::SetLoadData). " +
		"A flag is on or off, so an artifact cannot be held twice. The game caps what artifacts " +
		"add: strength, magic and defence at 99, hearts at 8, command slots at 8 " +
		"(CCaravanWork::CalcStatus).",
}

// Limits returns the limit of every value this package can change.
func Limits() []Limit {
	return []Limit{GilLimit, ArtifactLimit}
}
