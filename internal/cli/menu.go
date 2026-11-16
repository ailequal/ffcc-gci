package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/ailequal/ffcc-gci/internal/itemdb"
	"github.com/ailequal/ffcc-gci/internal/save"
)

// The menu runs the other commands, so the list that defines the commands
// cannot name its code: Go rejects that as an initialization cycle.
func init() { findCommand(commands, "menu").run = runMenu }

// menu is one run of the menu command.
//
// The menu holds no editing code of its own. It asks what to change,
// builds the command line of set-gil, add-artifact or add-item, shows it
// and runs it. An edit made from the menu is therefore checked and
// written exactly as one made with flags.
type menu struct {
	p      prompter
	stdout io.Writer
	stderr io.Writer
	path   string // the save file
}

// target is the character the menu is editing.
type target struct {
	slot      int // index of the slot, from 0
	character int // index of the character, from 0
}

func runMenu(c *command, args []string, stdout, stderr io.Writer) error {
	files, err := c.parse(c.flagSet(stderr), args, stdout)
	if err != nil {
		return err
	}
	if len(files) > 1 {
		return usagef("menu takes at most one save file, got %d", len(files))
	}

	p, err := newPrompter()
	if err != nil {
		return err
	}
	m := &menu{p: p, stdout: stdout, stderr: stderr}
	if len(files) == 1 {
		m.path = files[0]
	}

	err = m.run()
	if errors.Is(err, errBack) || errors.Is(err, errQuit) {
		return nil // the user left the menu
	}
	return err
}

func (m *menu) say(format string, args ...any) {
	_, _ = fmt.Fprintf(m.stdout, format+"\n", args...)
}

// run is the menu from the first question to the last.
func (m *menu) run() error {
	if m.path == "" {
		path, err := m.pickSave()
		if err != nil {
			return err
		}
		m.path = path
	}
	f, err := openSave(m.path)
	if err != nil {
		return err
	}
	// An absolute path, so that the command lines the menu shows can be
	// pasted in any directory.
	m.path = f.path
	if abs, err := filepath.Abs(f.path); err == nil {
		m.path = abs
	}

	// The path stays out of the frame: it can be wider than the terminal.
	m.p.frame(m.stdout, "ffcc-gci menu\nEsc goes back one step, Ctrl+C quits.\nClose Dolphin before editing.")
	m.say("Save: %s\n", m.path)

	for {
		t, only, err := m.pickTarget()
		if err != nil {
			return err
		}
		err = m.edit(t, only)
		if errors.Is(err, errBack) && !only {
			continue // back to the choice of character
		}
		return err
	}
}

// dolphinUserDirs returns the directories Dolphin may keep its user data
// in, as Dolphin itself picks them (UICommon::SetUserDirectory). Tests
// replace it.
var dolphinUserDirs = func() []string {
	var dirs []string
	add := func(base string, sub ...string) {
		if base != "" {
			dirs = append(dirs, filepath.Join(append([]string{base}, sub...)...))
		}
	}

	add(os.Getenv("DOLPHIN_EMU_USERPATH"))
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "windows":
		add(os.Getenv("APPDATA"), "Dolphin Emulator")
		add(home, "Documents", "Dolphin Emulator")
	case "darwin":
		add(home, "Library", "Application Support", "Dolphin")
	default:
		add(home, ".dolphin-emu")
		if data := os.Getenv("XDG_DATA_HOME"); data != "" {
			add(data, "dolphin-emu")
		} else {
			add(home, ".local", "share", "dolphin-emu")
		}
		add(home, ".var", "app", "org.DolphinEmu.dolphin-emu", "data", "dolphin-emu")
	}
	return dirs
}

// dolphinSaves returns the supported saves found in the European memory
// card folders of Dolphin, GC/EUR/Card A and Card B.
func dolphinSaves() []string {
	var saves []string
	for _, dir := range dolphinUserDirs() {
		found, _ := filepath.Glob(filepath.Join(dir, "GC", "EUR", "Card ?", "*.gci"))
		for _, path := range found {
			f, err := openSave(path)
			if err == nil && !slices.Contains(saves, f.path) {
				saves = append(saves, f.path)
			}
		}
	}
	return saves
}

// pickSave asks which save file to open: one of those found in Dolphin's
// folders, or any file the user browses to.
func (m *menu) pickSave() (string, error) {
	const browse = "Browse for another file..."

	if saves := dolphinSaves(); len(saves) > 0 {
		labels := make([]string, len(saves), len(saves)+1)
		for i, path := range saves {
			labels[i] = shortPath(path)
		}
		i, err := m.p.choose("Which save? These were found in Dolphin's folders.", append(labels, browse))
		if err != nil {
			return "", err
		}
		if i < len(saves) {
			return saves[i], nil
		}
	} else {
		m.say("No save of the game was found in Dolphin's usual folders. Pick the .gci file.\n")
	}

	dir, err := os.Getwd()
	if err != nil {
		dir = "."
	}
	return m.p.file(dir)
}

// shortPath returns path with the user's home directory written as "~",
// so that a long path fits a line of a list.
func shortPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" || runtime.GOOS == "windows" {
		return path
	}
	if rest, ok := strings.CutPrefix(path, home+string(filepath.Separator)); ok {
		return filepath.Join("~", rest)
	}
	return path
}

// editable opens the save and returns the slots that can be edited and,
// for each, the indices of its characters.
func (m *menu) editable() (f *saveFile, slots []int, characters map[int][]int, err error) {
	f, err = openSave(m.path)
	if err != nil {
		return nil, nil, nil, err
	}
	characters = make(map[int][]int)
	for si := range save.SlotCount {
		slot, err := f.data.Slot(si)
		if err != nil {
			continue // empty or not loadable
		}
		for ci := range save.CharacterCount {
			if c, err := slot.Character(ci); err == nil && c.Exists() {
				characters[si] = append(characters[si], ci)
			}
		}
		if len(characters[si]) > 0 {
			slots = append(slots, si)
		}
	}
	if len(slots) == 0 {
		return nil, nil, nil, fmt.Errorf("%s has no slot with a character to edit", m.path)
	}
	return f, slots, characters, nil
}

// pickTarget asks which character to edit. A question with one possible
// answer is not asked; only reports that there was no choice at all.
func (m *menu) pickTarget() (t target, only bool, err error) {
	f, slots, characters, err := m.editable()
	if err != nil {
		return t, false, err
	}
	only = len(slots) == 1 && len(characters[slots[0]]) == 1

	for {
		t.slot = slots[0]
		if len(slots) > 1 {
			labels := make([]string, len(slots))
			for i, si := range slots {
				slot, _ := f.data.Slot(si)
				hours, minutes := slot.PlayTime()
				labels[i] = fmt.Sprintf("Slot %d: town %s, play time %d:%02d, %s",
					si+1, slot.TownName(), hours, minutes, plural(len(characters[si]), "character"))
			}
			i, err := m.p.choose("Which slot?", labels)
			if err != nil {
				return t, only, err
			}
			t.slot = slots[i]
		}

		in := characters[t.slot]
		t.character = in[0]
		if len(in) == 1 {
			return t, only, nil
		}
		slot, _ := f.data.Slot(t.slot)
		labels := make([]string, len(in))
		for i, ci := range in {
			c, _ := slot.Character(ci)
			labels[i] = fmt.Sprintf("%d. %s (%s), %d gil", ci+1, c.Name(), c.Tribe(), c.Gil())
		}
		i, err := m.p.choose(fmt.Sprintf("Which character of slot %d?", t.slot+1), labels)
		if errors.Is(err, errBack) && len(slots) > 1 {
			continue // back to the choice of slot
		}
		if err != nil {
			return t, only, err
		}
		t.character = in[i]
		return t, only, nil
	}
}

// load reads the character from the file again, so that every question
// shows the save as it is now.
func (m *menu) load(t target) (*save.Character, error) {
	f, err := openSave(m.path)
	if err != nil {
		return nil, err
	}
	slot, err := f.data.Slot(t.slot)
	if err != nil {
		return nil, fmt.Errorf("slot %d: %w", t.slot+1, err)
	}
	return slot.Character(t.character)
}

// edit asks what to do with the character until the user goes back or
// quits. only says that the save has no other character to switch to.
func (m *menu) edit(t target, only bool) error {
	// Esc inside a change leaves the change, not the list of actions.
	change := func(ask func(target) error) func() error {
		return func() error {
			if err := ask(t); !errors.Is(err, errBack) {
				return err
			}
			return nil
		}
	}
	type action struct {
		label string
		do    func() error
	}
	actions := []action{
		{"Set gil", change(m.setGil)},
		{"Add an artifact", change(m.addArtifact)},
		{"Add an item", change(m.addItem)},
	}
	if !only {
		actions = append(actions, action{"Choose another character", func() error { return errBack }})
	}
	actions = append(actions,
		action{"Show the whole save", func() error { return m.command("info", m.path) }},
		action{"Show what can be edited", func() error { return m.command("limits") }},
		action{"Quit", func() error { return errQuit }},
	)
	labels := make([]string, len(actions))
	for i, a := range actions {
		labels[i] = a.label
	}

	for {
		c, err := m.load(t)
		if err != nil {
			return err
		}
		header := fmt.Sprintf("Slot %d, %s (%s): %d gil, %d of %d items, %s",
			t.slot+1, c.Name(), c.Tribe(), c.Gil(), c.ItemCount(), save.InventorySize,
			plural(len(c.Artifacts()), "artifact"))
		i, err := m.p.choose(header, labels)
		if err != nil {
			return err
		}
		if err := actions[i].do(); err != nil {
			return err
		}
	}
}

// number asks for a whole number that passes check. The question itself
// refuses any other answer and says why.
//
// start is what the answer field holds at first: a number to accept with
// Enter, or nothing.
func (m *menu) number(header, start string, check func(int) error) (int, error) {
	parse := func(text string) (int, error) {
		n, err := strconv.Atoi(strings.TrimSpace(text))
		if err != nil {
			return 0, fmt.Errorf("%q is not a whole number", text)
		}
		return n, check(n)
	}
	text, err := m.p.input(header, start, func(text string) error {
		_, err := parse(text)
		return err
	})
	if err != nil {
		return 0, err
	}
	// The same check again: the edit never rests on the prompter alone.
	return parse(text)
}

func (m *menu) setGil(t target) error {
	c, err := m.load(t)
	if err != nil {
		return err
	}
	header := fmt.Sprintf("Gil for %s, %d to %d (now %d)", c.Name(), save.GilLimit.Min, save.GilLimit.Max, c.Gil())
	gil, err := m.number(header, "", save.GilLimit.Check)
	if err != nil {
		return err
	}
	return m.apply(t, fmt.Sprintf("set the gil of %s to %d", c.Name(), gil),
		"set-gil", "-gil", strconv.Itoa(gil))
}

func (m *menu) addArtifact(t target) error {
	c, err := m.load(t)
	if err != nil {
		return err
	}
	var ids []int
	var labels []string
	for _, id := range artifactIDs() {
		if c.HasArtifact(id) {
			continue
		}
		desc, _ := itemdb.Description(id)
		ids = append(ids, id)
		labels = append(labels, fmt.Sprintf("%d  %-15s  %s", id, itemName(id), desc))
	}
	if len(ids) == 0 {
		m.say("%s already has every artifact.\n", c.Name())
		return nil
	}

	header := fmt.Sprintf("Artifact for %s, who has %d of %d", c.Name(), len(artifactIDs())-len(ids), len(artifactIDs()))
	i, err := m.p.search(header, labels)
	if err != nil {
		return err
	}
	return m.apply(t, fmt.Sprintf("grant %s the artifact %s", c.Name(), itemName(ids[i])),
		"add-artifact", "-artifact", strconv.Itoa(ids[i]))
}

func (m *menu) addItem(t target) error {
	c, err := m.load(t)
	if err != nil {
		return err
	}
	free := c.FreeSlots()
	if free == 0 {
		m.say("The inventory of %s is full: %d of %d items.\n", c.Name(), save.InventorySize, save.InventorySize)
		return nil
	}

	// A design the character already has, or could not forge, is not
	// offered: add-item would refuse it.
	var ids []int
	var labels []string
	for _, id := range addableIDs() {
		if save.IsDesign(id) && (c.HasItem(id) || checkDesign(id, 1, c) != nil) {
			continue
		}
		desc, _ := itemdb.Description(id)
		ids = append(ids, id)
		labels = append(labels, fmt.Sprintf("%d  %-15s  %s", id, itemName(id), desc))
	}
	header := fmt.Sprintf("Item for %s, who has %s", c.Name(), plural(free, "free inventory slot"))
	i, err := m.p.search(header, labels)
	if err != nil {
		return err
	}
	id, name := ids[i], itemName(ids[i])

	if save.IsDesign(id) {
		return m.apply(t, fmt.Sprintf("add the design %s to %s", name, c.Name()),
			"add-item", "-item", strconv.Itoa(id))
	}
	count, err := m.number(fmt.Sprintf("How many %s? 1 to %d, one inventory slot each", name, free), "1",
		func(n int) error {
			if n < 1 || n > free {
				return fmt.Errorf("the number must be between 1 and %d, got %d", free, n)
			}
			return nil
		})
	if err != nil {
		return err
	}
	return m.apply(t, fmt.Sprintf("add %d x %s to %s", count, name, c.Name()),
		"add-item", "-item", strconv.Itoa(id), "-count", strconv.Itoa(count))
}

// apply shows the edit and the command line that makes it, asks for
// confirmation and runs the command. what says what the edit does.
func (m *menu) apply(t target, what, name string, flags ...string) error {
	args := []string{name, "-slot", strconv.Itoa(t.slot + 1), "-char", strconv.Itoa(t.character + 1)}
	args = append(append(args, flags...), m.path)

	m.say("Slot %d: %s.\nThe same as: %s\n", t.slot+1, what, commandLine(args))
	yes, err := m.p.confirm("Write this change to the save?")
	if err != nil {
		return err
	}
	if !yes {
		m.say("Nothing written.\n")
		return nil
	}
	return m.command(args...)
}

// command runs one command of the program, as if it had been typed. A
// command that fails has said why, and the menu goes on.
func (m *menu) command(args ...string) error {
	Run(args, m.stdout, m.stderr)
	m.say("")
	return nil
}

// commandLine returns the command line that runs ffcc-gci with args,
// quoted so that it can be pasted into a shell.
func commandLine(args []string) string {
	parts := []string{"ffcc-gci"}
	for _, arg := range args {
		plain := arg != "" && strings.IndexFunc(arg, func(r rune) bool {
			return !strings.ContainsRune("-_./:=+,@%", r) && (r < '0' || r > '9') && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z')
		}) < 0
		switch {
		case plain:
			parts = append(parts, arg)
		case runtime.GOOS == "windows":
			parts = append(parts, `"`+arg+`"`)
		default:
			parts = append(parts, "'"+strings.ReplaceAll(arg, "'", `'\''`)+"'")
		}
	}
	return strings.Join(parts, " ")
}

// plural returns "1 thing" or "n things".
func plural(n int, thing string) string {
	if n == 1 {
		return "1 " + thing
	}
	return fmt.Sprintf("%d %ss", n, thing)
}
