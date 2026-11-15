package cli

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// catalog is a set of items the user can pick from by name or item ID.
type catalog struct {
	noun  string           // what one entry is called, such as "artifact"
	none  string           // error text for a name that matches nothing; takes the name
	ids   []int            // the entries, in listing order
	check func(int) error  // validates an item ID typed as a number
	name  func(int) string // the name of an entry
}

// find resolves what the user typed to an item ID: an item ID, an exact
// name, or a part of a name that matches a single entry. Names are
// compared without regard to case.
func (c catalog) find(sel string) (int, error) {
	if id, err := strconv.Atoi(sel); err == nil {
		if err := c.check(id); err != nil {
			return 0, err
		}
		return id, nil
	}

	want := strings.ToLower(strings.TrimSpace(sel))
	if want == "" {
		return 0, errors.New("the " + c.noun + " name is empty")
	}
	var partial []int
	for _, id := range c.ids {
		name := strings.ToLower(c.name(id))
		if name == want {
			return id, nil
		}
		if strings.Contains(name, want) {
			partial = append(partial, id)
		}
	}

	switch len(partial) {
	case 1:
		return partial[0], nil
	case 0:
		return 0, fmt.Errorf(c.none, sel)
	default:
		names := make([]string, len(partial))
		for i, id := range partial {
			names[i] = c.name(id)
		}
		return 0, fmt.Errorf("%q matches %d %ss, be more specific: %s", sel, len(partial), c.noun, strings.Join(names, ", "))
	}
}

// matching returns the entries whose name contains text, without regard
// to case. An empty text matches every entry.
func (c catalog) matching(text string) []int {
	text = strings.ToLower(text)
	var ids []int
	for _, id := range c.ids {
		if strings.Contains(strings.ToLower(c.name(id)), text) {
			ids = append(ids, id)
		}
	}
	return ids
}
