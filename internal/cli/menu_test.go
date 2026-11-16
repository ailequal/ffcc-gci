package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/ailequal/ffcc-gci/internal/itemdb"
	"github.com/ailequal/ffcc-gci/internal/save"
	"github.com/ailequal/ffcc-gci/internal/save/savetest"
	"github.com/charmbracelet/x/term"
)

// Answers a script gives in place of text: the keys that leave a prompt.
const (
	escKey = "<esc>"
	ctrlC  = "<ctrl-c>"
)

// script is a prompter that answers from a list, in order. An answer to
// a list is a text that exactly one of the options contains.
type script struct {
	t       *testing.T
	answers []string
	headers []string   // the header of every question asked
	lists   [][]string // the options of every list shown
	refused []string   // why each answer that was not accepted was refused
}

// useScript makes the menu ask its questions to a script with the given
// answers, and keeps it away from the saves of the machine's own Dolphin.
// The test fails if an answer is left over.
func useScript(t *testing.T, answers ...string) *script {
	t.Helper()
	s := &script{t: t, answers: answers}

	oldPrompter, oldDirs := newPrompter, dolphinUserDirs
	newPrompter = func() (prompter, error) { return s, nil }
	dolphinUserDirs = func() []string { return nil }
	t.Cleanup(func() {
		newPrompter, dolphinUserDirs = oldPrompter, oldDirs
		if len(s.answers) > 0 {
			t.Errorf("the menu never asked for the answers %q", s.answers)
		}
	})
	return s
}

func (s *script) next(header string) (string, error) {
	s.t.Helper()
	s.headers = append(s.headers, header)
	if len(s.answers) == 0 {
		s.t.Errorf("the menu asked %q and the script has no answer left", header)
		return "", errQuit
	}
	answer := s.answers[0]
	s.answers = s.answers[1:]
	switch answer {
	case escKey:
		return "", errBack
	case ctrlC:
		return "", errQuit
	}
	return answer, nil
}

func (s *script) choose(header string, options []string) (int, error) {
	s.t.Helper()
	s.lists = append(s.lists, options)
	// gum answers with the text of an option, so no two may be the same.
	if distinct := slices.Compact(slices.Sorted(slices.Values(options))); len(distinct) != len(options) {
		s.t.Errorf("the options of %q are not all different: %q", header, options)
	}
	answer, err := s.next(header)
	if err != nil {
		return 0, err
	}
	picked := -1
	for i, option := range options {
		if strings.Contains(option, answer) {
			if picked >= 0 {
				s.t.Fatalf("the answer %q to %q matches both %q and %q", answer, header, options[picked], option)
			}
			picked = i
		}
	}
	if picked < 0 {
		s.t.Fatalf("the answer %q to %q matches none of %q", answer, header, options)
	}
	return picked, nil
}

func (s *script) search(header string, options []string) (int, error) {
	s.t.Helper()
	return s.choose(header, options)
}

// input answers like the real question: an answer check refuses is not
// returned, and the next one is tried.
func (s *script) input(header, _ string, check func(string) error) (string, error) {
	for {
		answer, err := s.next(header)
		if err != nil {
			return "", err
		}
		if err := check(answer); err != nil {
			s.refused = append(s.refused, err.Error())
			continue
		}
		return answer, nil
	}
}

func (s *script) confirm(question string) (bool, error) {
	answer, err := s.next(question)
	return answer == "yes", err
}

func (s *script) file(dir string) (string, error) { return s.next("file in " + dir) }

func (s *script) frame(w io.Writer, text string) { _, _ = fmt.Fprintln(w, text) }

// offered reports whether any list the menu showed had an option that
// starts with prefix.
func (s *script) offered(prefix string) bool {
	for _, list := range s.lists {
		for _, option := range list {
			if strings.HasPrefix(option, prefix) {
				return true
			}
		}
	}
	return false
}

// asked reports whether the menu asked a question containing text.
func (s *script) asked(text string) bool {
	return slices.ContainsFunc(s.headers, func(h string) bool { return strings.Contains(h, text) })
}

func TestMenuSetGil(t *testing.T) {
	path, original := writeSample(t)
	fixClock(t)
	useScript(t, "Slot 1", "Alba", "Set gil", "5000", "yes", "Quit")

	got := run("menu", path)

	if got.code != ExitOK {
		t.Fatalf("exit code = %d, stderr = %q", got.code, got.stderr)
	}
	if gil := gilOf(t, path, 0, 0); gil != 5000 {
		t.Errorf("gil on disk = %d, want 5000", gil)
	}
	for _, want := range []string{
		"Slot 1: set the gil of Alba to 5000.",
		"ffcc-gci set-gil -slot 1 -char 1 -gil 5000 ",
		"Slot 1, Alba: gil 200 -> 5000",
		"Backup of the original: ",
	} {
		if !strings.Contains(got.stdout, want) {
			t.Errorf("stdout = %q, want it to contain %q", got.stdout, want)
		}
	}
	// The edit went through set-gil, so the original is backed up.
	backup := path + ".20261115-143005.bak"
	if !bytes.Equal(mustRead(t, backup), original) {
		t.Errorf("%s is not a copy of the original save", backup)
	}
}

func TestMenuAsksAgainAndWritesNothingWithoutConfirmation(t *testing.T) {
	path, original := writeSample(t)
	s := useScript(t, "Slot 1", "Alba", "Set gil", "lots", "100000000", "-1", "5", "no", "Quit")

	got := run("menu", path)

	if got.code != ExitOK {
		t.Fatalf("exit code = %d, stderr = %q", got.code, got.stderr)
	}
	wantRefused := []string{
		`"lots" is not a whole number`,
		"value out of range: gil must be between 0 and 99999999, got 100000000",
		"value out of range: gil must be between 0 and 99999999, got -1",
	}
	if !slices.Equal(s.refused, wantRefused) {
		t.Errorf("answers refused = %q, want %q", s.refused, wantRefused)
	}
	if !strings.Contains(got.stdout, "Nothing written.") {
		t.Errorf("stdout = %q, want it to say that nothing was written", got.stdout)
	}
	if !s.asked("Gil for Alba, 0 to 99999999 (now 200)") {
		t.Errorf("the question does not state the range: %q", s.headers)
	}
	if !bytes.Equal(mustRead(t, path), original) || len(backups(t, path)) != 0 {
		t.Error("the save was touched without confirmation")
	}
}

func TestMenuAddArtifact(t *testing.T) {
	path, _ := writeSample(t)
	// Bruno has the artifacts 160 and 183.
	s := useScript(t, "Slot 1", "Bruno", "Add an artifact", "Sun Pendant", "yes", "Quit")

	got := run("menu", path)

	if got.code != ExitOK {
		t.Fatalf("exit code = %d, stderr = %q", got.code, got.stderr)
	}
	if owned := artifactsOf(t, path, 0, 1); !slices.Equal(owned, []int{160, 183, 231}) {
		t.Errorf("artifacts on disk = %v, want 160, 183 and 231", owned)
	}
	if !strings.Contains(got.stdout, "ffcc-gci add-artifact -slot 1 -char 2 -artifact 231 ") {
		t.Errorf("stdout = %q, want it to show the command", got.stdout)
	}
	if !s.asked("Artifact for Bruno, who has 2 of 73") {
		t.Errorf("questions = %q, want the count of artifacts", s.headers)
	}
	if !s.offered("159  ") || !s.offered("231  Sun Pendant      Gain another heart") {
		t.Error("the list lacks artifacts Bruno does not have")
	}
	if s.offered("160  ") || s.offered("183  ") {
		t.Error("the list offers artifacts Bruno already has")
	}
}

func TestMenuAddItem(t *testing.T) {
	path, _ := writeSample(t)
	// Alba, a Clavat, has 3 items: 61 slots are free.
	s := useScript(t, "Slot 1", "Alba", "Add an item", "296  ", "0", "62", "2", "yes", "Quit")

	got := run("menu", path)

	if got.code != ExitOK {
		t.Fatalf("exit code = %d, stderr = %q", got.code, got.stderr)
	}
	inventory := inventoryOf(t, path, 0, 0)
	if n := len(inventory); n != 5 || inventory[3] != 296 || inventory[4] != 296 {
		t.Errorf("inventory on disk = %v, want two Mythril added to the three items", inventory)
	}
	wantRefused := []string{
		"the number must be between 1 and 61, got 0",
		"the number must be between 1 and 61, got 62",
	}
	if !slices.Equal(s.refused, wantRefused) {
		t.Errorf("answers refused = %q, want %q", s.refused, wantRefused)
	}
	for _, want := range []string{
		"Slot 1: add 2 x Mythril to Alba.",
		"ffcc-gci add-item -slot 1 -char 1 -item 296 -count 2 ",
	} {
		if !strings.Contains(got.stdout, want) {
			t.Errorf("stdout = %q, want it to contain %q", got.stdout, want)
		}
	}
	if !s.asked("Item for Alba, who has 61 free inventory slots") || !s.asked("How many Mythril? 1 to 61") {
		t.Errorf("questions = %q, want the free slots", s.headers)
	}

	// Only what add-item accepts for a Clavat is offered.
	for _, id := range addableIDs() {
		user, isDesign := itemdb.DesignFor(id)
		want := !isDesign || user == itemdb.ForAnyone || user == itemdb.ForClavats
		if got := s.offered(fmt.Sprintf("%d  ", id)); got != want {
			t.Errorf("item %d (%s) offered = %v, want %v", id, itemName(id), got, want)
		}
	}
}

func TestMenuAddDesignOnlyOnce(t *testing.T) {
	path, _ := writeSample(t)
	s := useScript(t, "Slot 1", "Alba", "Add an item", "413  ", "yes", "Add an item", escKey, "Quit")

	got := run("menu", path)

	if got.code != ExitOK {
		t.Fatalf("exit code = %d, stderr = %q", got.code, got.stderr)
	}
	if inventory := inventoryOf(t, path, 0, 0); !slices.Contains(inventory, 413) {
		t.Errorf("inventory on disk = %v, want the design 413", inventory)
	}
	// A design is added without asking how many.
	if s.asked("How many") {
		t.Errorf("questions = %q, want none about the number of designs", s.headers)
	}
	if !strings.Contains(got.stdout, "ffcc-gci add-item -slot 1 -char 1 -item 413 ") {
		t.Errorf("stdout = %q, want it to show the command", got.stdout)
	}
	// The second list of items no longer has the design.
	last := s.lists[len(s.lists)-2]
	if slices.ContainsFunc(last, func(o string) bool { return strings.HasPrefix(o, "413  ") }) {
		t.Error("a design the character has is offered again")
	}
}

func TestMenuWithOneCharacter(t *testing.T) {
	full := make([]int, save.InventorySize)
	for i := range full {
		full[i] = save.PhoenixDown
	}
	path := filepath.Join(t.TempDir(), "save.gci")
	content := savetest.GCI(nil, &savetest.Slot{
		Town: "Tipa", Key: 3,
		Characters: []savetest.Character{{Name: "Solo", Gil: 10, Items: full}},
	})
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	// No question about the slot or the character: there is no choice.
	s := useScript(t, "Add an item", "Quit")

	got := run("menu", path)

	if got.code != ExitOK {
		t.Fatalf("exit code = %d, stderr = %q", got.code, got.stderr)
	}
	if !s.asked("Slot 2, Solo (Clavat): 10 gil, 64 of 64 items, 0 artifacts") {
		t.Errorf("questions = %q, want the summary of the character", s.headers)
	}
	if !strings.Contains(got.stdout, "The inventory of Solo is full: 64 of 64 items.") {
		t.Errorf("stdout = %q, want it to say the inventory is full", got.stdout)
	}
	if s.offered("Choose another character") {
		t.Error("the menu offers another character when there is none")
	}
	if !bytes.Equal(mustRead(t, path), content) {
		t.Error("the save was changed")
	}
}

func TestMenuEscGoesBackOneStep(t *testing.T) {
	path, original := writeSample(t)
	s := useScript(t,
		"Slot 1", "Alba", escKey, // from the actions back to the slots
		"Slot 3", "1. Twin", "Set gil", escKey, // from the amount back to the actions
		"Add an artifact", escKey,
		"Add an item", "296  ", escKey,
		"Choose another character",
		"Slot 1", escKey, // from the characters back to the slots
		escKey, // from the slots out of the menu
	)

	got := run("menu", path)

	if got.code != ExitOK {
		t.Fatalf("exit code = %d, stderr = %q", got.code, got.stderr)
	}
	if !s.asked("Slot 3, Twin (Clavat): 1 gil") {
		t.Errorf("questions = %q, want the first of the two characters named Twin", s.headers)
	}
	if !bytes.Equal(mustRead(t, path), original) || len(backups(t, path)) != 0 {
		t.Error("the save was touched")
	}
}

func TestMenuShowsTheSaveAndTheLimits(t *testing.T) {
	path, _ := writeSample(t)
	useScript(t, "Slot 1", "Bruno", "Show the whole save", "Show what can be edited", ctrlC)

	got := run("menu", path)

	if got.code != ExitOK {
		t.Fatalf("exit code = %d, stderr = %q", got.code, got.stderr)
	}
	for _, want := range []string{"Slot 2: empty", "gil: 0 to 99999999", "Esc goes back one step, Ctrl+C quits.", "Save: " + path} {
		if !strings.Contains(got.stdout, want) {
			t.Errorf("stdout = %q, want it to contain %q", got.stdout, want)
		}
	}
}

func TestMenuFindsTheSavesOfDolphin(t *testing.T) {
	user := t.TempDir()
	card := filepath.Join(user, "GC", "EUR", "Card A")
	if err := os.MkdirAll(card, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(card, "01-GCCP-FFCC.gci")
	for name, content := range map[string][]byte{
		path:                               sampleGCI(),
		filepath.Join(card, "01-GZLP.gci"): []byte("the save of another game"),
		path + ".20261115-143005.bak":      sampleGCI(),
	} {
		if err := os.WriteFile(name, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	s := useScript(t, "01-GCCP-FFCC.gci", "Slot 1", "Alba", "Set gil", "7", "yes", "Quit")
	// The same folder twice: a save is offered once.
	dolphinUserDirs = func() []string { return []string{user, filepath.Join(user, "missing"), user} }

	got := run("menu")

	if got.code != ExitOK {
		t.Fatalf("exit code = %d, stderr = %q", got.code, got.stderr)
	}
	if n := len(s.lists[0]); n != 2 || !s.offered("Browse for another file...") {
		t.Errorf("saves offered = %q, want the one supported save and the choice to browse", s.lists[0])
	}
	if gil := gilOf(t, path, 0, 0); gil != 7 {
		t.Errorf("gil on disk = %d, want 7", gil)
	}
}

func TestShortPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the home directory is not shortened on Windows")
	}
	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("HOME", home)

	tests := []struct{ path, want string }{
		{filepath.Join(home, ".local", "share", "save.gci"), "~/.local/share/save.gci"},
		{home + "-other/save.gci", home + "-other/save.gci"},
		{"/media/card/save.gci", "/media/card/save.gci"},
	}
	for _, tt := range tests {
		if got := shortPath(tt.path); got != tt.want {
			t.Errorf("shortPath(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestMenuBrowsesForASave(t *testing.T) {
	path, _ := writeSample(t)
	s := useScript(t, path, "Slot 1", ctrlC)

	got := run("menu")

	if got.code != ExitOK {
		t.Fatalf("exit code = %d, stderr = %q", got.code, got.stderr)
	}
	if !strings.Contains(got.stdout, "No save of the game was found") || !s.asked("file in ") {
		t.Errorf("stdout = %q, questions = %q, want the file to be asked for", got.stdout, s.headers)
	}
}

func TestMenuRefuses(t *testing.T) {
	dir := t.TempDir()
	junk := filepath.Join(dir, "junk.gci")
	if err := os.WriteFile(junk, []byte("not a save"), 0o644); err != nil {
		t.Fatal(err)
	}
	empty := filepath.Join(dir, "empty.gci")
	if err := os.WriteFile(empty, savetest.GCI(), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStderr string
	}{
		{name: "two files", args: []string{"menu", junk, junk}, wantCode: ExitUsage, wantStderr: "at most one save file"},
		{name: "not a save", args: []string{"menu", junk}, wantCode: ExitError, wantStderr: "junk.gci"},
		{name: "missing file", args: []string{"menu", filepath.Join(dir, "none.gci")}, wantCode: ExitError, wantStderr: "no such file"},
		{name: "no character", args: []string{"menu", empty}, wantCode: ExitError, wantStderr: "has no slot with a character to edit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useScript(t) // no question is asked

			got := run(tt.args...)

			if got.code != tt.wantCode || !strings.Contains(got.stderr, tt.wantStderr) {
				t.Errorf("exit code = %d, stderr = %q, want %d and %q", got.code, got.stderr, tt.wantCode, tt.wantStderr)
			}
		})
	}
}

func TestMenuNeedsATerminal(t *testing.T) {
	if term.IsTerminal(os.Stdin.Fd()) && term.IsTerminal(os.Stdout.Fd()) {
		t.Skip("the test runs in a terminal, where the menu would open")
	}
	path, original := writeSample(t)

	got := run("menu", path)

	if got.code != ExitError || !strings.Contains(got.stderr, "in a terminal, and this is not one") {
		t.Errorf("exit code = %d, stderr = %q, want %d and the reason", got.code, got.stderr, ExitError)
	}
	if !bytes.Equal(mustRead(t, path), original) {
		t.Error("the save was changed")
	}
}

func TestCommandLine(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("quoting for a POSIX shell")
	}
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"limits"}, "ffcc-gci limits"},
		{[]string{"set-gil", "-gil", "5", "/saves/01-GCCP-FFCC.gci"}, "ffcc-gci set-gil -gil 5 /saves/01-GCCP-FFCC.gci"},
		{[]string{"info", "/GC/EUR/Card A/save.gci"}, "ffcc-gci info '/GC/EUR/Card A/save.gci'"},
		{[]string{"info", "it's.gci", ""}, `ffcc-gci info 'it'\''s.gci' ''`},
		{[]string{"info", "$HOME/*.gci"}, "ffcc-gci info '$HOME/*.gci'"},
	}
	for _, tt := range tests {
		if got := commandLine(tt.args); got != tt.want {
			t.Errorf("commandLine(%q) = %s, want %s", tt.args, got, tt.want)
		}
	}
}
