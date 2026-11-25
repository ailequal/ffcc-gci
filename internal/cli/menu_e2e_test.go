package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// The tests in this file run the built program in a real terminal and
// press keys in it, which is the only way to see the menu as a user does:
// that every question appears, takes the keys typed and leaves the screen
// clean for the next one.
//
// The terminal is a tmux session of its own, so they need tmux and are
// skipped without it. Set FFCC_GCI_E2E=require to make that a failure.

// e2eDir holds the program built for the terminal tests, once it has
// been built.
var e2eDir string

// e2eBinary builds the program once for all the terminal tests.
var e2eBinary = sync.OnceValues(func() (string, error) {
	dir, err := os.MkdirTemp("", "ffcc-gci-e2e-")
	if err != nil {
		return "", err
	}
	e2eDir = dir
	path := filepath.Join(dir, "ffcc-gci")
	build := exec.Command("go", "build", "-o", path, "github.com/ailequal/ffcc-gci/cmd/ffcc-gci")
	if out, err := build.CombinedOutput(); err != nil {
		return "", fmt.Errorf("go build: %w\n%s", err, out)
	}
	return path, nil
})

// TestMain removes the program built for the terminal tests.
func TestMain(m *testing.M) {
	code := m.Run()
	if e2eDir != "" {
		_ = os.RemoveAll(e2eDir)
	}
	os.Exit(code)
}

// screen is a terminal with the program running in it.
type screen struct {
	t      *testing.T
	socket string // path of the socket of this test's own tmux server
}

// shellQuote quotes arg for a POSIX shell.
func shellQuote(arg string) string {
	return "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'"
}

// openTerminal starts the program in a new terminal, with dir as its
// working directory and its home. The program therefore finds no save of
// the machine's own Dolphin. env holds further variables, as "NAME=value".
// When the program ends, the terminal shows "EXIT=" and its exit code.
func openTerminal(t *testing.T, dir string, env []string, args ...string) *screen {
	t.Helper()
	required := os.Getenv("FFCC_GCI_E2E") == "require"
	switch _, err := exec.LookPath("tmux"); {
	case testing.Short():
		t.Skip("the terminal tests are skipped in short mode")
	case runtime.GOOS == "windows":
		t.Skip("the terminal tests need tmux")
	case err != nil && required:
		t.Fatal("tmux was not found, and FFCC_GCI_E2E=require is set")
	case err != nil:
		t.Skip("tmux was not found")
	}
	binary, err := e2eBinary()
	if err != nil {
		t.Fatal(err)
	}

	command := "env -u XDG_DATA_HOME -u DOLPHIN_EMU_USERPATH -u NO_COLOR HOME=" + shellQuote(dir)
	for _, variable := range env {
		command += " " + shellQuote(variable)
	}
	command += " " + shellQuote(binary)
	for _, arg := range args {
		command += " " + shellQuote(arg)
	}
	command += `; echo "EXIT=$?"; sleep 600`

	// The socket gets a directory of its own, with a short path as
	// sockets need, that is removed with the server: tmux leaves the
	// socket of a killed server behind.
	socketDir, err := os.MkdirTemp("", "ffcc-gci-tmux-")
	if err != nil {
		t.Fatal(err)
	}
	s := &screen{t: t, socket: filepath.Join(socketDir, "s")}
	t.Cleanup(func() {
		_ = exec.Command("tmux", "-S", s.socket, "kill-server").Run()
		_ = os.RemoveAll(socketDir)
	})
	// -f /dev/null: the settings of the user's own tmux play no part.
	s.tmux("-f", "/dev/null", "new-session", "-d", "-x", "160", "-y", "45", "-c", dir, "sh", "-c", command)
	return s
}

func (s *screen) tmux(args ...string) string {
	s.t.Helper()
	out, err := exec.Command("tmux", append([]string{"-S", s.socket}, args...)...).CombinedOutput()
	if err != nil {
		s.t.Fatalf("tmux %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// text returns what the terminal shows, without the spaces at the ends
// of its lines.
func (s *screen) text() string {
	s.t.Helper()
	lines := strings.Split(s.tmux("capture-pane", "-p", "-J"), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

// waitUntil waits for the terminal to show something that passes ok.
func (s *screen) waitUntil(what string, ok func(text string) bool) string {
	s.t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for {
		text := s.text()
		if ok(text) {
			return text
		}
		if time.Now().After(deadline) {
			s.t.Fatalf("the terminal never showed %s. It shows:\n%s", what, text)
		}
		time.Sleep(30 * time.Millisecond)
	}
}

// see waits for the terminal to show every one of wants.
func (s *screen) see(wants ...string) string {
	s.t.Helper()
	return s.waitUntil(fmt.Sprintf("%q", wants), func(text string) bool {
		for _, want := range wants {
			if !strings.Contains(text, want) {
				return false
			}
		}
		return true
	})
}

// gone waits for the terminal to show none of texts any more.
func (s *screen) gone(texts ...string) {
	s.t.Helper()
	s.waitUntil(fmt.Sprintf("the last of %q", texts), func(text string) bool {
		for _, unwanted := range texts {
			if strings.Contains(text, unwanted) {
				return false
			}
		}
		return true
	})
}

// press sends keys by their tmux names, such as "Enter", "Down" or "C-c".
func (s *screen) press(keys ...string) {
	s.t.Helper()
	s.tmux(append([]string{"send-keys"}, keys...)...)
}

// typeText types text as it is.
func (s *screen) typeText(text string) {
	s.t.Helper()
	s.tmux("send-keys", "-l", text)
}

// titled is the line a question's title is drawn on.
func titled(title string) string { return "┃ " + title }

// TestMenuInATerminal makes every kind of edit through the menu, typing
// as a user would, and then reads the save back.
func TestMenuInATerminal(t *testing.T) {
	dir := tempDir(t)
	path := filepath.Join(dir, "save.gci")
	original := sampleGCI()
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatal(err)
	}
	s := openTerminal(t, dir, nil, "menu", path)

	// The slot and the character.
	s.see("ffcc-gci menu", "Save: "+path, titled("Which slot?"),
		"> Slot 1: town Tipa, play time 1:15, 2 characters", "  Slot 3: town Tipa, play time 0:00, 2 characters")
	s.press("Enter")
	s.see(titled("Which character of slot 1?"), "> 1. Alba (Clavat), 200 gil", "  2. Bruno (Selkie), 346 gil")
	s.gone(titled("Which slot?"))
	s.press("Down")
	s.see("> 2. Bruno (Selkie), 346 gil")
	s.press("Enter")
	const actions = "┃ Slot 1, Bruno (Selkie): "
	s.see(actions+"346 gil, 0 of 64 items, 2 artifacts", "> Set gil", "  Quit")
	s.gone("Which character")

	// Gil: a number out of range, a text, then a confirmation left at "No".
	s.press("Enter")
	s.see(titled("Gil for Bruno, 0 to 99999999 (now 346)"))
	s.gone(actions)
	s.typeText("100000000")
	s.see("> 100000000")
	s.press("Enter")
	s.see("gil must be between 0 and 99999999, got 100000000")
	for range len("100000000") {
		s.press("BSpace")
	}
	s.typeText("lots")
	s.see("> lots")
	s.press("Enter")
	s.see(`"lots" is not a whole number`)
	for range len("lots") {
		s.press("BSpace")
	}
	s.typeText("5000")
	s.see("> 5000")
	s.press("Enter")
	s.see("Slot 1: set the gil of Bruno to 5000.", "The same as: ffcc-gci set-gil -slot 1 -char 2 -gil 5000 "+path,
		titled("Write this change to the save?"))
	s.gone("Gil for Bruno")
	s.press("Enter") // "No" is the default
	s.see("Nothing written.", actions+"346 gil")
	s.gone("Write this change")
	if !bytes.Equal(mustRead(t, path), original) {
		t.Fatal("the save was changed without a yes")
	}

	// Gil again, confirmed.
	s.press("Enter")
	s.see(titled("Gil for Bruno"))
	s.typeText("5000")
	s.see("> 5000")
	s.press("Enter")
	s.see(titled("Write this change to the save?"))
	s.press("y")
	s.see("Slot 1, Bruno: gil 346 -> 5000", "Backup of the original: "+path, actions+"5000 gil, 0 of 64 items, 2 artifacts")
	s.gone("Write this change")

	// An artifact, found by typing a part of its name.
	s.press("Down")
	s.see("> Add an artifact")
	s.press("Enter")
	s.see("Artifact for Bruno, who has 2 of 73", "> 159  Shuriken")
	s.gone(actions)
	if text := s.text(); strings.Contains(text, "160  Maneater") {
		t.Errorf("the list offers an artifact Bruno has:\n%s", text)
	}
	s.typeText("sun")
	s.see("┃ /sun", "> 231  Sun Pendant      Gain another heart")
	s.gone("159  Shuriken")
	s.press("Enter")
	s.see("Slot 1: grant Bruno the artifact Sun Pendant.", "ffcc-gci add-artifact -slot 1 -char 2 -artifact 231 "+path,
		titled("Write this change to the save?"))
	s.gone("Artifact for Bruno")
	s.press("Left") // move to "Yes"
	s.press("Enter")
	s.see("Slot 1, Bruno: added artifact Sun Pendant (hearts)", actions+"5000 gil, 0 of 64 items, 3 artifacts")
	s.gone("Write this change")

	// Items: several of one, moving in the narrowed list.
	s.press("Down")
	s.see("> Add an artifact")
	s.press("Down")
	s.see("> Add an item")
	s.press("Enter")
	s.see("Item for Bruno, who has 64 free inventory slots", "> 293  Phoenix Down")
	s.typeText("mythril")
	s.see("┃ /mythril", "> 296  Mythril          Material for equipment")
	s.gone("293  Phoenix Down")
	s.press("Enter")
	s.see(titled("How many Mythril? 1 to 64, one inventory slot each"), "> 1")
	s.gone("Item for Bruno")
	s.press("BSpace")
	s.typeText("65")
	s.see("> 65")
	s.press("Enter")
	s.see("the number must be between 1 and 64, got 65")
	s.press("BSpace", "BSpace")
	s.typeText("3")
	s.see("> 3")
	s.press("Enter")
	s.see("Slot 1: add 3 x Mythril to Bruno.", "ffcc-gci add-item -slot 1 -char 2 -item 296 -count 3 "+path,
		titled("Write this change to the save?"))
	s.press("y")
	s.see("Slot 1, Bruno: added 3 x Mythril (items 0 -> 3 of 64)", actions+"5000 gil, 3 of 64 items, 3 artifacts")
	s.gone("Write this change")

	// Esc goes back one step at a time; Ctrl+C leaves the menu.
	s.press("Escape")
	s.see(titled("Which slot?"))
	s.gone(actions, "> Set gil")
	s.press("Down")
	s.see("> Slot 3")
	s.press("Enter")
	s.see(titled("Which character of slot 3?"), "> 1. Twin (Clavat), 1 gil", "  2. twin (Clavat), 2 gil")
	s.press("Escape")
	s.see(titled("Which slot?"))
	s.gone("Which character")
	s.press("C-c")
	final := s.see("EXIT=0")
	// Nothing of a question is left on the screen.
	if strings.Contains(final, "┃") {
		t.Errorf("a question was left on the screen:\n%s", final)
	}

	// The save holds the three edits and nothing else changed for Alba.
	if gil := gilOf(t, path, 0, 1); gil != 5000 {
		t.Errorf("gil of Bruno on disk = %d, want 5000", gil)
	}
	if owned := artifactsOf(t, path, 0, 1); !slices.Equal(owned, []int{160, 183, 231}) {
		t.Errorf("artifacts of Bruno on disk = %v, want 160, 183 and 231", owned)
	}
	if items := inventoryOf(t, path, 0, 1); !slices.Equal(items, []int{296, 296, 296}) {
		t.Errorf("items of Bruno on disk = %v, want three Mythril", items)
	}
	if gil := gilOf(t, path, 0, 0); gil != 200 {
		t.Errorf("gil of Alba on disk = %d, want it untouched at 200", gil)
	}
	if n := len(backups(t, path)); n != 3 {
		t.Errorf("%d backups were written, want one for each of the 3 edits", n)
	}
}

// TestMenuFindsTheSaveInATerminal starts the menu without a file, once
// with a save in Dolphin's folder and once with none.
func TestMenuFindsTheSaveInATerminal(t *testing.T) {
	t.Run("in Dolphin's folder", func(t *testing.T) {
		dir := tempDir(t)
		// Where Dolphin keeps its user data below the home directory
		// when no variable says otherwise.
		user := filepath.Join(".local", "share", "dolphin-emu")
		if runtime.GOOS == "darwin" {
			user = filepath.Join("Library", "Application Support", "Dolphin")
		}
		card := filepath.Join(dir, user, "GC", "EUR", "Card A")
		if err := os.MkdirAll(card, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(card, "01-GCCP-FFCC.gci"), sampleGCI(), 0o644); err != nil {
			t.Fatal(err)
		}
		s := openTerminal(t, dir, nil, "menu")

		s.see(titled("Which save? These were found in Dolphin's folders."),
			"> "+filepath.Join("~", user, "GC", "EUR", "Card A", "01-GCCP-FFCC.gci"), "  Browse for another file...")
		s.press("Enter")
		s.see("Save: "+filepath.Join(card, "01-GCCP-FFCC.gci"), titled("Which slot?"))
		s.gone("Which save?")
		s.press("C-c")
		s.see("EXIT=0")
	})

	t.Run("browsing", func(t *testing.T) {
		dir := tempDir(t)
		for name, content := range map[string][]byte{
			"a-notes.txt": []byte("notes"),
			"save.gci":    sampleGCI(),
		} {
			if err := os.WriteFile(filepath.Join(dir, name), content, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		s := openTerminal(t, dir, nil, "menu")

		s.see("No save of the game was found in Dolphin's usual folders.", titled("Pick the save file"), "a-notes.txt", "save.gci")
		s.press("Down")
		s.press("Enter")
		s.see("Save: "+filepath.Join(dir, "save.gci"), titled("Which slot?"))
		s.gone("Pick the save file")
		s.press("Escape")
		final := s.see("EXIT=0")
		if strings.Contains(final, "┃") {
			t.Errorf("a question was left on the screen:\n%s", final)
		}
	})
}

// TestMenuRefusesInATerminal checks what the user sees when the menu
// cannot start.
func TestMenuRefusesInATerminal(t *testing.T) {
	dir := t.TempDir()
	junk := filepath.Join(dir, "junk.gci")
	if err := os.WriteFile(junk, []byte("not a save"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := openTerminal(t, dir, nil, "menu", junk)

	final := s.see("EXIT=1")
	if !strings.Contains(final, "ffcc-gci: "+junk) || strings.Contains(final, "┃") {
		t.Errorf("the terminal shows:\n%s\nwant the reason and no question", final)
	}
}
