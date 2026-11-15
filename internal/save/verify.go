package save

import (
	"bytes"
	"fmt"
)

// VerifyEdit checks that after differs from before only in the ways an
// edit of one slot is allowed to: nothing outside the slot changed, the
// slot is still one the game would load, and inside the decoded slot only
// the checksum and the allowed ranges changed.
//
// It is the last check before an edited save is written to disk.
func VerifyEdit(before, after *Data, slot int, allowed ...Range) error {
	if err := checkSlotIndex(slot); err != nil {
		return err
	}

	start := BannerSize + slot*SlotSize
	end := start + SlotSize
	if !bytes.Equal(before.b[:start], after.b[:start]) || !bytes.Equal(before.b[end:], after.b[end:]) {
		return fmt.Errorf("edit changed data outside slot %d", slot)
	}

	old := Decode(before.stored(slot))
	edited := Decode(after.stored(slot))
	if err := validate(edited); err != nil {
		return fmt.Errorf("edited slot is not loadable: %w", err)
	}

	allowed = append(allowed, Range{Start: offCRC, End: offCRC + 4})
	for off := range edited {
		if old[off] != edited[off] && !inAny(off, allowed) {
			return fmt.Errorf("edit changed unexpected byte at offset %#x of slot %d", off, slot)
		}
	}
	return nil
}

func inAny(off int, ranges []Range) bool {
	for _, r := range ranges {
		if off >= r.Start && off < r.End {
			return true
		}
	}
	return false
}
