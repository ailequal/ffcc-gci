package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"unicode"

	"github.com/ailequal/ffcc-gci/internal/save"
	"github.com/ailequal/ffcc-gci/internal/save/savetest"
	"github.com/charmbracelet/x/term"
)

// The documents written by hand are tested too: their examples must run,
// the messages they quote must be the ones the program prints, and their
// links must lead somewhere. docs/USAGE.md is generated, and compared
// with the program's help in TestUsageDocIsUpToDate.

// TestDocExamplesAreAccepted runs every ffcc-gci command line shown in
// the README and in the guide against a sample save, so that neither can
// show a command the program refuses.
func TestDocExamplesAreAccepted(t *testing.T) {
	docs := []struct {
		path string
		min  int // fewer would mean the document changed its form and nothing is tested
	}{
		{"../../README.md", 10},
		{"../../docs/GUIDE.md", 25},
	}
	// A comment after a command, as in "ffcc-gci help    # the list".
	comment := regexp.MustCompile(`\s{2,}#.*$`)

	for _, doc := range docs {
		text, err := os.ReadFile(doc.path)
		if err != nil {
			t.Fatal(err)
		}
		var lines []string
		fenced := false
		for _, line := range strings.Split(string(text), "\n") {
			if strings.HasPrefix(line, "```") {
				fenced = !fenced
				continue
			}
			if fenced && strings.HasPrefix(line, "ffcc-gci ") {
				lines = append(lines, comment.ReplaceAllString(line, ""))
			}
		}
		if len(lines) < doc.min {
			t.Fatalf("%s: found %d command lines, want at least %d", doc.path, len(lines), doc.min)
		}

		for _, line := range lines {
			t.Run(filepath.Base(doc.path)+": "+line, func(t *testing.T) {
				args := splitCommandLine(line)[1:]
				if args[0] == "menu" {
					useScript(t, ctrlC) // the menu opens and the user leaves it
				}
				dir := t.TempDir()
				// The examples are about Alba, a Clavat in slot 3.
				content := savetest.GCI(nil, nil, &savetest.Slot{
					Town: "Tipa", Key: 5,
					Characters: []savetest.Character{
						{Name: "Alba", Tribe: 0, Gil: 200, Items: []int{293, 296}},
						{Name: "Bruno", Tribe: 3, Gil: 346},
					},
				})
				path := filepath.Join(dir, "save.gci")
				if err := os.WriteFile(path, content, 0o644); err != nil {
					t.Fatal(err)
				}
				// Whatever the document calls the save, it is the sample;
				// the file -o writes is a new one.
				for i, arg := range args {
					switch {
					case arg == "edited.gci":
						args[i] = filepath.Join(dir, arg)
					case strings.HasSuffix(arg, ".gci"):
						args[i] = path
					}
				}
				if got := run(args...); got.code != ExitOK {
					t.Errorf("exit code = %d, stderr = %q", got.code, got.stderr)
				}
			})
		}
	}
}

// TestGuideQuotesTheProgram checks the messages the guide quotes in its
// table of refusals against what the program prints.
func TestGuideQuotesTheProgram(t *testing.T) {
	guide, err := os.ReadFile("../../docs/GUIDE.md")
	if err != nil {
		t.Fatal(err)
	}
	full := make([]int, save.InventorySize)
	for i := range full {
		full[i] = save.PhoenixDown
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "save.gci")
	content := savetest.GCI(nil, nil, &savetest.Slot{
		Town: "Tipa", Key: 5,
		Characters: []savetest.Character{
			{Name: "Alba", Tribe: 0, Gil: 200, Items: []int{413}, Artifacts: []int{72}},
			{Name: "Bruno", Tribe: 3, Gil: 346},
			{Name: "Full", Tribe: 1, Items: full},
			{Name: "Twin"}, {Name: "twin"},
		},
	})
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	junk := filepath.Join(dir, "junk.gci")
	if err := os.WriteFile(junk, []byte("not a save"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := []string{"-slot", "3", "-char", "Alba", path}

	tests := []struct {
		quote string // as the guide has it
		args  []string
	}{
		{`slot 1: slot is empty`, []string{"set-gil", "-slot", "1", "-char", "Alba", "-gil", "5", path}},
		{`there is no character named "Carla"; available: 1 (Alba), 2 (Bruno)`, []string{"set-gil", "-slot", "3", "-char", "Carla", "-gil", "5", path}},
		{`2 characters are named "Twin"; pick one by number`, []string{"set-gil", "-slot", "3", "-char", "Twin", "-gil", "5", path}},
		{`gil must be between 0 and 99999999`, append([]string{"set-gil", "-gil", "100000000"}, target...)},
		{`"ring" matches 8 artifacts, be more specific`, append([]string{"add-artifact", "-artifact", "ring"}, target...)},
		{`no item that can be added is named "Copper Sword"`, append([]string{"add-item", "-item", "Copper Sword"}, target...)},
		{`item ID 1 cannot be added; the allowed IDs are 293, 294-351, 381-392, 401-493`, append([]string{"add-item", "-item", "1"}, target...)},
		{`has 0 free inventory slots of 64, not enough for 1`, []string{"add-item", "-slot", "3", "-char", "Full", "-item", "Mythril", path}},
		{`is a design for Clavats and Bruno is a Selkie, who could not forge it`, []string{"add-item", "-slot", "3", "-char", "Bruno", "-item", "Mythril Shield", path}},
		{`designs for men or for women are not supported`, append([]string{"add-item", "-item", "Zeal Kit"}, target...)},
		{`is a design and can be added only once`, append([]string{"add-item", "-item", "Mythril Armour", "-count", "2"}, target...)},
		{`not a valid GCI file`, []string{"info", junk}},
		{`could not write the new file: `, append([]string{"set-gil", "-gil", "5", "-o", junk}, target...)},
		{`no such file or directory`, []string{"info", filepath.Join(dir, "Card A", "save.gci")}},
		{`the menu asks its questions in a terminal, and this is not one`, []string{"menu", path}},
		{`gil is already 200. Nothing written.`, append([]string{"set-gil", "-gil", "200"}, target...)},
		{`already has Sun Pendant. An artifact cannot be held twice. Nothing written.`, append([]string{"add-artifact", "-artifact", "Sun Pendant"}, target...)},
		{`already has the design Iron Armour. A design is added only once. Nothing written.`, append([]string{"add-item", "-item", "Iron Armour"}, target...)},
	}
	for _, tt := range tests {
		t.Run(tt.quote, func(t *testing.T) {
			if tt.args[0] == "menu" && term.IsTerminal(os.Stdin.Fd()) && term.IsTerminal(os.Stdout.Fd()) {
				t.Skip("the test runs in a terminal, where the menu would open")
			}
			if runtime.GOOS == "windows" && strings.Contains(tt.quote, "no such file") {
				t.Skip("Windows words this message differently")
			}
			if !strings.Contains(string(guide), tt.quote) {
				t.Errorf("the guide does not quote %q", tt.quote)
			}
			got := run(tt.args...)
			if !strings.Contains(got.stdout+got.stderr, tt.quote) {
				t.Errorf("the program does not print %q, but:\n%s%s", tt.quote, got.stdout, got.stderr)
			}
		})
	}
	// Messages the tests cannot provoke through the command line.
	for _, quote := range []string{
		"changed on disk while it was being edited; is Dolphin running?",
		save.ErrUnsupported.Error(),
		save.ErrCorruptSlot.Error(),
	} {
		if !strings.Contains(string(guide), quote) {
			t.Errorf("the guide does not quote %q", quote)
		}
	}
}

// TestDocLinksResolve follows every link between the documents: the file
// must exist, and so must the heading a link points at.
func TestDocLinksResolve(t *testing.T) {
	root := "../.."
	docs := []string{"README.md", "CHANGELOG.md", "docs/GUIDE.md", "docs/USAGE.md", "docs/SAVE_FORMAT.md"}
	link := regexp.MustCompile(`\]\(([^)\s]+)\)`)
	heading := regexp.MustCompile(`(?m)^#{1,6} (.+)$`)

	// anchors returns the anchors GitHub gives the headings of a document:
	// lower case, punctuation dropped, spaces turned into hyphens.
	anchors := func(text string) map[string]bool {
		found := make(map[string]bool)
		fenced := regexp.MustCompile("(?s)```.*?```").ReplaceAllString(text, "")
		for _, m := range heading.FindAllStringSubmatch(fenced, -1) {
			var b strings.Builder
			for _, r := range strings.ToLower(m[1]) {
				switch {
				case r == ' ':
					b.WriteRune('-')
				case r == '-' || r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r):
					b.WriteRune(r)
				}
			}
			found[b.String()] = true
		}
		return found
	}

	checked := 0
	for _, doc := range docs {
		text, err := os.ReadFile(filepath.Join(root, doc))
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range link.FindAllStringSubmatch(string(text), -1) {
			target := m[1]
			if strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
				continue // other sites are not followed
			}
			file, anchor, _ := strings.Cut(target, "#")
			path := filepath.Join(root, doc)
			if file != "" {
				path = filepath.Join(root, filepath.Dir(doc), file)
			}
			linked, err := os.ReadFile(path)
			if err != nil {
				t.Errorf("%s links to %s, which does not exist", doc, target)
				continue
			}
			if anchor != "" && !anchors(string(linked))[anchor] {
				t.Errorf("%s links to %s, but %s has no such heading", doc, target, filepath.Base(path))
			}
			checked++
		}
	}
	if checked < 20 {
		t.Errorf("followed %d links, want at least 20: the documents changed their form", checked)
	}
}
