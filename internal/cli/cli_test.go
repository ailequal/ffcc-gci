package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ailequal/ffcc-gci/internal/gci"
	"github.com/ailequal/ffcc-gci/internal/itemdb"
	"github.com/ailequal/ffcc-gci/internal/save"
	"github.com/ailequal/ffcc-gci/internal/save/savetest"
)

// result is what one invocation of the command line produced.
type result struct {
	code   int
	stdout string
	stderr string
}

func run(args ...string) result {
	var stdout, stderr bytes.Buffer
	code := Run(args, &stdout, &stderr)
	return result{code: code, stdout: stdout.String(), stderr: stderr.String()}
}

func sampleGCI() []byte {
	return savetest.GCI(
		&savetest.Slot{
			Town: "Tipa", Frames: 1500 * 75, Key: 186,
			Characters: []savetest.Character{
				{Name: "Alba", Tribe: 0, Gil: 200, Items: []int{0x01, 0x125, 0x125}},
				{Name: "Bruno", Tribe: 3, Gil: 346, Artifacts: []int{1, 24}},
			},
		},
		nil,
		&savetest.Slot{
			Town: "Tipa", Key: 7,
			Characters: []savetest.Character{{Name: "Twin", Gil: 1}, {Name: "twin", Gil: 2}},
		},
	)
}

// tempDir returns a new temporary directory by its real path, which is the
// one the program prints: it resolves the symbolic links of a save's path,
// and on macOS the temporary directory is reached through one.
func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

// writeSample writes the sample save to a temporary directory and returns
// its path and contents.
func writeSample(t *testing.T) (path string, content []byte) {
	t.Helper()
	content = sampleGCI()
	path = filepath.Join(tempDir(t), "save.gci")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	return path, content
}

// gilOf reads the gil of a character from a save file on disk.
func gilOf(t *testing.T, path string, slot, character int) int {
	t.Helper()
	f, err := openSave(path)
	if err != nil {
		t.Fatalf("openSave: %v", err)
	}
	s, err := f.data.Slot(slot)
	if err != nil {
		t.Fatalf("Slot(%d): %v", slot, err)
	}
	c, err := s.Character(character)
	if err != nil {
		t.Fatalf("Character(%d): %v", character, err)
	}
	return c.Gil()
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// backups returns the backup files next to the save at path.
func backups(t *testing.T, path string) []string {
	t.Helper()
	found, err := filepath.Glob(path + ".*.bak")
	if err != nil {
		t.Fatal(err)
	}
	return found
}

func fixClock(t *testing.T) {
	t.Helper()
	old := now
	now = func() time.Time { return time.Date(2026, 11, 15, 14, 30, 5, 0, time.UTC) }
	t.Cleanup(func() { now = old })
}

func TestRunBasics(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{name: "no arguments", args: nil, wantCode: ExitUsage, wantStderr: "Usage:"},
		{name: "help", args: []string{"help"}, wantCode: ExitOK, wantStdout: "set-gil"},
		{name: "help flag", args: []string{"--help"}, wantCode: ExitOK, wantStdout: "Usage:"},
		{name: "version", args: []string{"version"}, wantCode: ExitOK, wantStdout: "ffcc-gci " + Version + " (" + runtime.Version() + ", " + runtime.GOOS + "/" + runtime.GOARCH + ")\n"},
		{name: "unknown command", args: []string{"bogus"}, wantCode: ExitUsage, wantStderr: `unknown command "bogus"`},
		{name: "command help", args: []string{"set-gil", "-h"}, wantCode: ExitOK, wantStdout: "Close Dolphin"},
		{name: "help for a command", args: []string{"help", "add-item"}, wantCode: ExitOK, wantStdout: "ffcc-gci add-item -slot <1-4>"},
		{name: "help for a builtin", args: []string{"help", "help"}, wantCode: ExitOK, wantStdout: "-markdown"},
		{name: "help for an unknown command", args: []string{"help", "bogus"}, wantCode: ExitUsage, wantStderr: `there is no command "bogus"`},
		{name: "help with two modes", args: []string{"help", "-all", "-markdown"}, wantCode: ExitUsage, wantStderr: "one command, -all or -markdown"},
		{name: "unknown flag", args: []string{"info", "-bogus"}, wantCode: ExitUsage, wantStderr: "bogus"},
		{name: "info without file", args: []string{"info"}, wantCode: ExitUsage, wantStderr: "exactly one save file"},
		{name: "info missing file", args: []string{"info", "/nonexistent/save.gci"}, wantCode: ExitError, wantStderr: "no such file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := run(tt.args...)

			if got.code != tt.wantCode {
				t.Errorf("exit code = %d, want %d", got.code, tt.wantCode)
			}
			if !strings.Contains(got.stdout, tt.wantStdout) {
				t.Errorf("stdout = %q, want it to contain %q", got.stdout, tt.wantStdout)
			}
			if !strings.Contains(got.stderr, tt.wantStderr) {
				t.Errorf("stderr = %q, want it to contain %q", got.stderr, tt.wantStderr)
			}
		})
	}
}

func TestInfo(t *testing.T) {
	path, content := writeSample(t)

	got := run("info", path)

	if got.code != ExitOK {
		t.Fatalf("exit code = %d, stderr = %q", got.code, got.stderr)
	}
	for _, want := range []string{
		"save.gci: Final Fantasy Crystal Chronicles (PAL)",
		"Slot 1: town Tipa, play time 1:15",
		"1. Alba (Clavat, in party)",
		"Gil        200",
		"Hearts     4 of 4",
		"Stats      strength 10, magic 11, defence 12\n",
		"Hearts and stats are the numbers the game last stored",
		"Items      3 of 64: Copper Sword, Phoenix Down x2",
		"Artifacts  0\n",
		"2. Bruno (Selkie, in party)",
		"Items      0 of 64\n",
		"Artifacts  2: Maneater, Mage Masher",
		"Slot 2: empty",
		"Slot 3: town Tipa, play time 0:00",
		"Slot 4: empty",
	} {
		if !strings.Contains(got.stdout, want) {
			t.Errorf("output lacks %q\n%s", want, got.stdout)
		}
	}
	if !bytes.Equal(mustRead(t, path), content) {
		t.Error("info modified the save file")
	}
}

func TestInfoReportsCorruptSlot(t *testing.T) {
	path, content := writeSample(t)
	content[gci.HeaderSize+save.BannerSize+0x2000] ^= 0x01
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	got := run("info", path)

	if got.code != ExitOK || !strings.Contains(got.stdout, "Slot 1: not loadable") || !strings.Contains(got.stdout, "Slot 3: town Tipa") {
		t.Errorf("exit code = %d, output:\n%s", got.code, got.stdout)
	}
}

func TestInfoRejectsOtherFiles(t *testing.T) {
	usa := sampleGCI()
	copy(usa, "GCCE")

	tests := []struct {
		name    string
		content []byte
		want    string
	}{
		{name: "not a GCI", content: []byte("hello"), want: "not a valid GCI file"},
		{name: "truncated", content: sampleGCI()[:1000], want: "not a valid GCI file"},
		{name: "USA version", content: usa, want: "only PAL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "save.gci")
			if err := os.WriteFile(path, tt.content, 0o644); err != nil {
				t.Fatal(err)
			}

			got := run("info", path)

			if got.code != ExitError || !strings.Contains(got.stderr, tt.want) {
				t.Errorf("exit code = %d, stderr = %q, want code %d and %q", got.code, got.stderr, ExitError, tt.want)
			}
		})
	}
}

func TestSetGilInPlace(t *testing.T) {
	fixClock(t)
	path, original := writeSample(t)

	got := run("set-gil", "-slot", "1", "-char", "bruno", "-gil", "5000", path)

	if got.code != ExitOK {
		t.Fatalf("exit code = %d, stderr = %q", got.code, got.stderr)
	}
	if want := "Slot 1, Bruno: gil 346 -> 5000"; !strings.Contains(got.stdout, want) {
		t.Errorf("stdout = %q, want it to contain %q", got.stdout, want)
	}

	if gil := gilOf(t, path, 0, 1); gil != 5000 {
		t.Errorf("Bruno's gil = %d, want 5000", gil)
	}
	if gil := gilOf(t, path, 0, 0); gil != 200 {
		t.Errorf("Alba's gil = %d, want it untouched at 200", gil)
	}
	if gil := gilOf(t, path, 2, 0); gil != 1 {
		t.Errorf("slot 3 gil = %d, want it untouched at 1", gil)
	}

	backup := path + ".20261115-143005.bak"
	if !bytes.Equal(mustRead(t, backup), original) {
		t.Error("the backup is not identical to the original save")
	}
	if !strings.Contains(got.stdout, backup) {
		t.Errorf("stdout = %q, want it to name the backup %q", got.stdout, backup)
	}

	// Only the save and its backup may be left in the directory.
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Errorf("directory holds %d files, want 2 (save and backup)", len(entries))
	}
}

// TestEditsInTheSameSecondKeepEveryBackup makes several edits while the
// clock stands still, as a script that chains commands does. The name of
// a backup carries the time to the second, so each further backup needs
// a name of its own.
func TestEditsInTheSameSecondKeepEveryBackup(t *testing.T) {
	fixClock(t)
	path, original := writeSample(t)
	stamp := path + ".20261115-143005"

	var before [][]byte // the save as each edit found it
	for i, gil := range []string{"1000", "2000", "3000"} {
		before = append(before, mustRead(t, path))
		got := run("set-gil", "-slot", "1", "-char", "Alba", "-gil", gil, path)
		if got.code != ExitOK {
			t.Fatalf("edit %d: exit code = %d, stderr = %q", i+1, got.code, got.stderr)
		}
	}

	if gil := gilOf(t, path, 0, 0); gil != 3000 {
		t.Errorf("gil = %d, want 3000 from the last edit", gil)
	}
	for i, name := range []string{stamp + ".bak", stamp + "-2.bak", stamp + "-3.bak"} {
		if !bytes.Equal(mustRead(t, name), before[i]) {
			t.Errorf("%s is not the save as edit %d found it", filepath.Base(name), i+1)
		}
	}
	if !bytes.Equal(before[0], original) {
		t.Error("the first backup is not the original save")
	}
	if n := len(backups(t, path)); n != 3 {
		t.Errorf("%d backups were written, want 3", n)
	}
}

func TestSetGilToNewFile(t *testing.T) {
	path, original := writeSample(t)
	out := filepath.Join(t.TempDir(), "edited.gci")

	// Flags may also follow the file name.
	got := run("set-gil", path, "-slot", "1", "-char", "1", "-gil", "99999999", "-o", out)

	if got.code != ExitOK {
		t.Fatalf("exit code = %d, stderr = %q", got.code, got.stderr)
	}
	if gil := gilOf(t, out, 0, 0); gil != save.MaxGil {
		t.Errorf("gil in the new file = %d, want %d", gil, save.MaxGil)
	}
	if !bytes.Equal(mustRead(t, path), original) {
		t.Error("the original save was modified")
	}

	// A second run must not overwrite the file it wrote.
	again := run("set-gil", path, "-slot", "1", "-char", "1", "-gil", "7", "-o", out)
	if again.code != ExitError || !strings.Contains(again.stderr, "exists") {
		t.Errorf("overwriting -o: exit code = %d, stderr = %q", again.code, again.stderr)
	}
	// An edit that was not written is not announced.
	if again.stdout != "" {
		t.Errorf("stdout of the refused run = %q, want nothing", again.stdout)
	}
	if gil := gilOf(t, out, 0, 0); gil != save.MaxGil {
		t.Errorf("gil in the new file = %d after the refused run, want %d", gil, save.MaxGil)
	}
}

// TestSetGilAcceptsBothEndsOfTheRange sets the lowest and the highest
// value the limit allows.
func TestSetGilAcceptsBothEndsOfTheRange(t *testing.T) {
	for _, gil := range []int{save.GilLimit.Min, save.GilLimit.Max} {
		path, _ := writeSample(t)

		got := run("set-gil", "-slot", "1", "-char", "Alba", "-gil", fmt.Sprint(gil), path)

		if got.code != ExitOK {
			t.Errorf("gil %d: exit code = %d, stderr = %q", gil, got.code, got.stderr)
		}
		if after := gilOf(t, path, 0, 0); after != gil {
			t.Errorf("gil = %d, want %d", after, gil)
		}
	}
}

func TestLimits(t *testing.T) {
	got := run("limits")

	if got.code != ExitOK {
		t.Fatalf("exit code = %d, stderr = %q", got.code, got.stderr)
	}
	// The output is wrapped, so compare with line breaks removed.
	flat := strings.Join(strings.Fields(got.stdout), " ")
	for _, l := range save.Limits() {
		want := fmt.Sprintf("%s: %s", l.Name, rangeText(l))
		if !strings.Contains(flat, want) || !strings.Contains(flat, l.Source) {
			t.Errorf("output lacks the %q limit or its source\n%s", l.Name, got.stdout)
		}
	}
	for _, line := range strings.Split(got.stdout, "\n") {
		if len(line) > helpWidth {
			t.Errorf("line is %d characters wide, want at most %d: %q", len(line), helpWidth, line)
		}
	}
	for _, want := range []string{"slot: 1 to 4", "character: 1 to 8"} {
		if !strings.Contains(got.stdout, want) {
			t.Errorf("output lacks %q", want)
		}
	}

	if extra := run("limits", "unexpected"); extra.code != ExitUsage {
		t.Errorf("limits with an argument: exit code = %d, want %d", extra.code, ExitUsage)
	}
}

func TestSetGilHelpStatesTheRange(t *testing.T) {
	got := run("set-gil", "-h")

	want := fmt.Sprintf("%d to %d", save.GilLimit.Min, save.GilLimit.Max)
	if got.code != ExitOK || !strings.Contains(got.stdout, want) {
		t.Errorf("exit code = %d, help = %q, want it to state %q", got.code, got.stdout, want)
	}
}

func TestSetGilSameValueWritesNothing(t *testing.T) {
	path, original := writeSample(t)

	got := run("set-gil", "-slot", "1", "-char", "Alba", "-gil", "200", path)

	if got.code != ExitOK || !strings.Contains(got.stdout, "Nothing written") {
		t.Errorf("exit code = %d, stdout = %q", got.code, got.stdout)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || !bytes.Equal(mustRead(t, path), original) {
		t.Error("the save was rewritten or a backup was made")
	}
}

func TestSetGilRefuses(t *testing.T) {
	tests := []struct {
		name     string
		args     []string // the save path is appended
		wantCode int
		want     string
	}{
		{name: "missing slot", args: []string{"-char", "Alba", "-gil", "1"}, wantCode: ExitUsage, want: "needs -slot"},
		{name: "missing char", args: []string{"-slot", "1", "-gil", "1"}, wantCode: ExitUsage, want: "needs -char"},
		{name: "missing gil", args: []string{"-slot", "1", "-char", "Alba"}, wantCode: ExitUsage, want: "needs -gil"},
		{name: "slot out of range", args: []string{"-slot", "5", "-char", "Alba", "-gil", "1"}, wantCode: ExitUsage, want: "between 1 and 4"},
		{name: "negative gil", args: []string{"-slot", "1", "-char", "Alba", "-gil", "-1"}, wantCode: ExitUsage, want: "gil must be between 0 and 99999999, got -1"},
		{name: "gil above the cap", args: []string{"-slot", "1", "-char", "Alba", "-gil", "100000000"}, wantCode: ExitUsage, want: "gil must be between 0 and 99999999, got 100000000"},
		{name: "gil not a number", args: []string{"-slot", "1", "-char", "Alba", "-gil", "lots"}, wantCode: ExitUsage, want: "invalid value"},
		{name: "empty slot", args: []string{"-slot", "2", "-char", "Alba", "-gil", "1"}, wantCode: ExitError, want: "slot is empty"},
		{name: "unknown name", args: []string{"-slot", "1", "-char", "Zed", "-gil", "1"}, wantCode: ExitError, want: `no character named "Zed"; available: 1 (Alba), 2 (Bruno)`},
		{name: "unused number", args: []string{"-slot", "1", "-char", "3", "-gil", "1"}, wantCode: ExitError, want: "no character number 3"},
		{name: "ambiguous name", args: []string{"-slot", "3", "-char", "TWIN", "-gil", "9"}, wantCode: ExitError, want: "pick one by number"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, original := writeSample(t)

			got := run(append(append([]string{"set-gil"}, tt.args...), path)...)

			if got.code != tt.wantCode || !strings.Contains(got.stderr, tt.want) {
				t.Errorf("exit code = %d, stderr = %q, want code %d and %q", got.code, got.stderr, tt.wantCode, tt.want)
			}
			entries, err := os.ReadDir(filepath.Dir(path))
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 || !bytes.Equal(mustRead(t, path), original) {
				t.Error("a refused edit left a changed save or extra files behind")
			}
		})
	}
}

func TestSetGilRefusesCorruptSlot(t *testing.T) {
	path, content := writeSample(t)
	content[gci.HeaderSize+save.BannerSize+0x2000] ^= 0x01
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	got := run("set-gil", "-slot", "1", "-char", "Alba", "-gil", "1", path)

	if got.code != ExitError || !strings.Contains(got.stderr, "does not hold a valid save") {
		t.Errorf("exit code = %d, stderr = %q", got.code, got.stderr)
	}
	if !bytes.Equal(mustRead(t, path), content) {
		t.Error("a refused edit changed the file")
	}
}

func TestWriteInPlaceRefusesWhenFileChanged(t *testing.T) {
	path, _ := writeSample(t)
	f, err := openSave(path)
	if err != nil {
		t.Fatal(err)
	}

	// Something else, such as a running Dolphin, rewrites the save
	// between reading it and writing it back.
	changed := sampleGCI()
	changed[gci.HeaderSize] ^= 0x01
	if err := os.WriteFile(path, changed, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := f.writeInPlace(f.raw); err == nil || !strings.Contains(err.Error(), "changed on disk") {
		t.Errorf("writeInPlace error = %v, want a changed-on-disk error", err)
	}
	if !bytes.Equal(mustRead(t, path), changed) {
		t.Error("writeInPlace overwrote a file that had changed")
	}
}

func TestSetGilThroughSymlink(t *testing.T) {
	fixClock(t)
	path, _ := writeSample(t)
	link := filepath.Join(t.TempDir(), "link.gci")
	if err := os.Symlink(path, link); err != nil {
		t.Skipf("symbolic links are not available: %v", err)
	}

	got := run("set-gil", "-slot", "1", "-char", "Alba", "-gil", "42", link)

	if got.code != ExitOK {
		t.Fatalf("exit code = %d, stderr = %q", got.code, got.stderr)
	}
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Error("the symbolic link was replaced by a regular file")
	}
	if gil := gilOf(t, path, 0, 0); gil != 42 {
		t.Errorf("gil in the link's target = %d, want 42", gil)
	}
}

// artifactsOf reads the artifacts of a character from a save file on disk.
func artifactsOf(t *testing.T, path string, slot, character int) []int {
	t.Helper()
	f, err := openSave(path)
	if err != nil {
		t.Fatalf("openSave: %v", err)
	}
	s, err := f.data.Slot(slot)
	if err != nil {
		t.Fatalf("Slot(%d): %v", slot, err)
	}
	c, err := s.Character(character)
	if err != nil {
		t.Fatalf("Character(%d): %v", character, err)
	}
	return c.Artifacts()
}

func TestArtifactsList(t *testing.T) {
	got := run("artifacts")

	if got.code != ExitOK {
		t.Fatalf("exit code = %d, stderr = %q", got.code, got.stderr)
	}
	for _, want := range []string{
		"Strength\n  159  Shuriken         Strength +1\n",
		"Magic\n  182  Drgn. Whisker    Magic +1\n",
		"Defence\n  204  Main Gauche      Defence +2\n",
		"Command slots\n  219  Moogle Pocket    Gain another command slot\n",
		"Spell rings\n  223  Fire Ring        Cast Fire at any time\n",
		"Hearts\n  228  Earth Pendant    Gain another heart\n",
		"  231  Sun Pendant      Gain another heart\n",
	} {
		if !strings.Contains(got.stdout, want) {
			t.Errorf("output lacks %q\n%s", want, got.stdout)
		}
	}
	if n := strings.Count(got.stdout, "\n  "); n != 73 {
		t.Errorf("listed %d artifacts, want 73", n)
	}
	if strings.Contains(got.stdout, "Equip.") {
		t.Error("the list includes an unnamed placeholder")
	}
}

func TestArtifactsFilter(t *testing.T) {
	got := run("artifacts", "pendant")
	if got.code != ExitOK || strings.Count(got.stdout, "Pendant") != 4 || strings.Contains(got.stdout, "Shuriken") {
		t.Errorf("exit code = %d, output:\n%s", got.code, got.stdout)
	}

	none := run("artifacts", "zzz")
	if none.code != ExitError || !strings.Contains(none.stderr, `no artifact name contains "zzz"`) {
		t.Errorf("exit code = %d, stderr = %q", none.code, none.stderr)
	}
}

func TestAddArtifact(t *testing.T) {
	tests := []struct {
		name     string
		artifact string
		wantID   int
		wantLine string
	}{
		{name: "exact name", artifact: "Sun Pendant", wantID: 0xE7, wantLine: "Slot 1, Alba: added artifact Sun Pendant (hearts)"},
		{name: "name in another case", artifact: "mage masher", wantID: 0xB7, wantLine: "added artifact Mage Masher (magic)"},
		{name: "unique part of a name", artifact: "kaiser", wantID: 0xA3, wantLine: "added artifact Kaiser Knckls. (strength)"},
		{name: "exact name that is also part of others", artifact: "Kris", wantID: 0xC6, wantLine: "added artifact Kris (magic)"},
		{name: "item ID", artifact: "219", wantID: 0xDB, wantLine: "added artifact Moogle Pocket (command slots)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixClock(t)
			path, original := writeSample(t)

			got := run("add-artifact", "-slot", "1", "-char", "Alba", "-artifact", tt.artifact, path)

			if got.code != ExitOK {
				t.Fatalf("exit code = %d, stderr = %q", got.code, got.stderr)
			}
			if !strings.Contains(got.stdout, tt.wantLine) {
				t.Errorf("stdout = %q, want it to contain %q", got.stdout, tt.wantLine)
			}
			if !strings.Contains(got.stdout, "until the game saves") {
				t.Errorf("stdout = %q, want the note about stored stats", got.stdout)
			}
			if owned := artifactsOf(t, path, 0, 0); len(owned) != 1 || owned[0] != tt.wantID {
				t.Errorf("Alba's artifacts = %#x, want [%#x]", owned, tt.wantID)
			}
			if owned := artifactsOf(t, path, 0, 1); len(owned) != 2 {
				t.Errorf("Bruno's artifacts = %#x, want his two untouched", owned)
			}
			if gil := gilOf(t, path, 0, 0); gil != 200 {
				t.Errorf("Alba's gil = %d, want it untouched at 200", gil)
			}
			if !bytes.Equal(mustRead(t, path+".20261115-143005.bak"), original) {
				t.Error("the backup is not identical to the original save")
			}
		})
	}
}

func TestAddArtifactAlreadyOwnedWritesNothing(t *testing.T) {
	path, original := writeSample(t)

	// Bruno owns Maneater and Mage Masher in the sample.
	got := run("add-artifact", "-slot", "1", "-char", "Bruno", "-artifact", "Maneater", path)

	if got.code != ExitOK || !strings.Contains(got.stdout, "already has Maneater") || !strings.Contains(got.stdout, "Nothing written") {
		t.Errorf("exit code = %d, stdout = %q", got.code, got.stdout)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || !bytes.Equal(mustRead(t, path), original) {
		t.Error("the save was rewritten or a backup was made")
	}
}

func TestAddArtifactToNewFile(t *testing.T) {
	path, original := writeSample(t)
	out := filepath.Join(t.TempDir(), "edited.gci")

	got := run("add-artifact", "-slot", "1", "-char", "2", "-artifact", "Ribbon", "-o", out, path)

	if got.code != ExitOK {
		t.Fatalf("exit code = %d, stderr = %q", got.code, got.stderr)
	}
	if owned := artifactsOf(t, out, 0, 1); len(owned) != 3 {
		t.Errorf("artifacts in the new file = %#x, want three", owned)
	}
	if !bytes.Equal(mustRead(t, path), original) {
		t.Error("the original save was modified")
	}
}

func TestAddArtifactRefuses(t *testing.T) {
	tests := []struct {
		name     string
		args     []string // the save path is appended
		wantCode int
		want     string
	}{
		{name: "missing artifact", args: []string{"-slot", "1", "-char", "Alba"}, wantCode: ExitUsage, want: "needs -artifact"},
		{name: "missing slot", args: []string{"-char", "Alba", "-artifact", "Ribbon"}, wantCode: ExitUsage, want: "needs -slot"},
		{name: "unknown name", args: []string{"-slot", "1", "-char", "Alba", "-artifact", "Excalibur"}, wantCode: ExitUsage, want: `no artifact is named "Excalibur"`},
		{name: "ambiguous name", args: []string{"-slot", "1", "-char", "Alba", "-artifact", "pendant"}, wantCode: ExitUsage, want: "matches 4 artifacts, be more specific: Earth Pendant, Moon Pendant, Star Pendant, Sun Pendant"},
		{name: "placeholder by name", args: []string{"-slot", "1", "-char", "Alba", "-artifact", "Equip. 187"}, wantCode: ExitUsage, want: "no artifact is named"},
		{name: "placeholder by ID", args: []string{"-slot", "1", "-char", "Alba", "-artifact", "232"}, wantCode: ExitUsage, want: "artifact item ID must be between 159 and 231, got 232"},
		{name: "ID below the range", args: []string{"-slot", "1", "-char", "Alba", "-artifact", "158"}, wantCode: ExitUsage, want: "artifact item ID must be between 159 and 231, got 158"},
		{name: "non-artifact item ID", args: []string{"-slot", "1", "-char", "Alba", "-artifact", "293"}, wantCode: ExitUsage, want: "between 159 and 231"},
		{name: "empty slot", args: []string{"-slot", "4", "-char", "Alba", "-artifact", "Ribbon"}, wantCode: ExitError, want: "slot is empty"},
		{name: "unknown character", args: []string{"-slot", "1", "-char", "Zed", "-artifact", "Ribbon"}, wantCode: ExitError, want: `no character named "Zed"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, original := writeSample(t)

			got := run(append(append([]string{"add-artifact"}, tt.args...), path)...)

			if got.code != tt.wantCode || !strings.Contains(got.stderr, tt.want) {
				t.Errorf("exit code = %d, stderr = %q, want code %d and %q", got.code, got.stderr, tt.wantCode, tt.want)
			}
			entries, err := os.ReadDir(filepath.Dir(path))
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 || !bytes.Equal(mustRead(t, path), original) {
				t.Error("a refused edit left a changed save or extra files behind")
			}
		})
	}
}

// TestEveryGrantableArtifactIsNamed makes sure the artifact range holds
// only real, uniquely named artifacts with a known effect.
func TestEveryGrantableArtifactIsNamed(t *testing.T) {
	seen := make(map[string]bool)
	for id := save.ArtifactLimit.Min; id <= save.ArtifactLimit.Max; id++ {
		name := itemName(id)
		if name == "" || strings.HasPrefix(name, "Equip.") {
			t.Errorf("item %d has no real name: %q", id, name)
		}
		if seen[strings.ToLower(name)] {
			t.Errorf("artifact name %q is used twice", name)
		}
		seen[strings.ToLower(name)] = true

		if got, err := artifactCatalog().find(name); err != nil || got != id {
			t.Errorf("find(%q) = %d, %v; want %d", name, got, err, id)
		}
	}
	if len(seen) != 73 {
		t.Errorf("%d artifacts in range, want 73", len(seen))
	}
}

// inventoryOf reads the used inventory slots of a character from a save
// file on disk, in slot order.
func inventoryOf(t *testing.T, path string, slot, character int) []int {
	t.Helper()
	f, err := openSave(path)
	if err != nil {
		t.Fatalf("openSave: %v", err)
	}
	s, err := f.data.Slot(slot)
	if err != nil {
		t.Fatalf("Slot(%d): %v", slot, err)
	}
	c, err := s.Character(character)
	if err != nil {
		t.Fatalf("Character(%d): %v", character, err)
	}
	var held []int
	for _, id := range c.Inventory() {
		if id != save.EmptyItem {
			held = append(held, id)
		}
	}
	if len(held) != c.ItemCount() {
		t.Fatalf("%d items held but the count field says %d", len(held), c.ItemCount())
	}
	return held
}

func TestItemsList(t *testing.T) {
	got := run("items")

	if got.code != ExitOK {
		t.Fatalf("exit code = %d, stderr = %q", got.code, got.stderr)
	}
	for _, want := range []string{
		"Phoenix Down\n  293  Phoenix Down\n",
		"Materials\n  294  Bronze\n",
		"  351  Remedy\n\nFood\n  381  Striped Apple\n",
		"  392  Strange Liquid\n\nDesigns\n  401  Novice's Wpn.    Design for anyone\n",
		"  411  Lunar Wpn.       Design for Selkies\n",
		"  493  Ring of Invin.   Design for anyone\n",
	} {
		if !strings.Contains(got.stdout, want) {
			t.Errorf("output lacks %q\n%s", want, got.stdout)
		}
	}
	if n := strings.Count(got.stdout, "\n  "); n != 164 {
		t.Errorf("listed %d items, want 164", n)
	}
	for _, banned := range []string{"Extra", "???", "design 9", "Equip.", "Fruit Seed", "Veg. Seed", "Wheat Seed", "Flower Seed", "Strange Seed", "Shella", "Bandana", "Goddess", "Ultimite", "Copper Sword", "Item Gil"} {
		if strings.Contains(got.stdout, banned) {
			t.Errorf("the list includes %q, which must not be addable", banned)
		}
	}

	filtered := run("items", "mythril")
	if filtered.code != ExitOK || !strings.Contains(filtered.stdout, "  296  Mythril\n") || strings.Contains(filtered.stdout, "Bronze\n") {
		t.Errorf("items mythril: exit code = %d, output:\n%s", filtered.code, filtered.stdout)
	}
	none := run("items", "racket")
	if none.code != ExitError || !strings.Contains(none.stderr, "no item that can be added") {
		t.Errorf("items racket: exit code = %d, stderr = %q", none.code, none.stderr)
	}
}

// TestEveryAddableItemIsDescribedByTheGame makes sure the item ranges hold
// only items the game itself describes as one of the allowed kinds, each
// with a unique name.
func TestEveryAddableItemIsDescribedByTheGame(t *testing.T) {
	seen := make(map[string]bool)
	ids := addableIDs()
	for _, id := range ids {
		name := itemName(id)
		if itemdb.KindOf(id) == itemdb.KindOther {
			desc, _ := itemdb.Description(id)
			t.Errorf("item %d (%s) is addable but the game describes it as %q", id, name, desc)
		}
		if seen[strings.ToLower(name)] {
			t.Errorf("item name %q is used twice", name)
		}
		seen[strings.ToLower(name)] = true

		if got, err := itemCatalog().find(name); err != nil || got != id {
			t.Errorf("find(%q) = %d, %v; want %d", name, got, err, id)
		}
	}
	if len(ids) != 164 {
		t.Errorf("%d addable items, want 164", len(ids))
	}

	// Each range must be exactly one kind.
	kinds := []itemdb.Kind{itemdb.KindPhoenixDown, itemdb.KindMaterial, itemdb.KindFood, itemdb.KindDesign}
	for i, l := range save.ItemLimits {
		for id := l.Min; id <= l.Max; id++ {
			if got := itemdb.KindOf(id); got != kinds[i] {
				t.Errorf("item %d in the %q range is %v, want %v", id, l.Name, got, kinds[i])
			}
		}
	}
}

func TestAddItem(t *testing.T) {
	tests := []struct {
		name     string
		char     string
		args     []string
		wantHeld []int
		wantLine string
	}{
		{name: "material by name", char: "Alba", args: []string{"-item", "Mythril"},
			wantHeld: []int{0x01, 0x125, 0x125, 296}, wantLine: "Slot 1, Alba: added Mythril (items 3 -> 4 of 64)"},
		{name: "several of one food", char: "Alba", args: []string{"-item", "striped apple", "-count", "3"},
			wantHeld: []int{0x01, 0x125, 0x125, 381, 381, 381}, wantLine: "added 3 x Striped Apple (items 3 -> 6 of 64)"},
		{name: "another Phoenix Down by ID", char: "Alba", args: []string{"-item", "293"},
			wantHeld: []int{0x01, 0x125, 0x125, 293}, wantLine: "added Phoenix Down (items 3 -> 4 of 64)"},
		{name: "unique part of a name", char: "Bruno", args: []string{"-item", "orichal"},
			wantHeld: []int{297}, wantLine: "Slot 1, Bruno: added Orichalcum (items 0 -> 1 of 64)"},
		{name: "design for anyone", char: "Alba", args: []string{"-item", "Novice's Wpn."},
			wantHeld: []int{0x01, 0x125, 0x125, 401}, wantLine: "added Novice's Wpn. (items 3 -> 4 of 64)"},
		{name: "design for the character's tribe", char: "Bruno", args: []string{"-item", "Lunar Wpn."},
			wantHeld: []int{411}, wantLine: "added Lunar Wpn. (items 0 -> 1 of 64)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixClock(t)
			path, original := writeSample(t)
			character := map[string]int{"Alba": 0, "Bruno": 1}[tt.char]

			args := append([]string{"add-item", "-slot", "1", "-char", tt.char}, tt.args...)
			got := run(append(args, path)...)

			if got.code != ExitOK {
				t.Fatalf("exit code = %d, stderr = %q", got.code, got.stderr)
			}
			if !strings.Contains(got.stdout, tt.wantLine) {
				t.Errorf("stdout = %q, want it to contain %q", got.stdout, tt.wantLine)
			}
			if held := inventoryOf(t, path, 0, character); !slices.Equal(held, tt.wantHeld) {
				t.Errorf("inventory = %v, want %v", held, tt.wantHeld)
			}
			other := 1 - character
			if held := inventoryOf(t, path, 0, other); len(held) != []int{3, 0}[other] {
				t.Errorf("the other character's inventory changed: %v", held)
			}
			if gil := gilOf(t, path, 0, character); gil != []int{200, 346}[character] {
				t.Errorf("gil = %d, want it untouched", gil)
			}
			if !bytes.Equal(mustRead(t, path+".20261115-143005.bak"), original) {
				t.Error("the backup is not identical to the original save")
			}
		})
	}
}

func TestAddItemDesignOnlyOnce(t *testing.T) {
	path, _ := writeSample(t)

	if got := run("add-item", "-slot", "1", "-char", "Alba", "-item", "Bronze Armour", path); got.code != ExitOK {
		t.Fatalf("first add: exit code = %d, stderr = %q", got.code, got.stderr)
	}
	edited := mustRead(t, path)
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}

	got := run("add-item", "-slot", "1", "-char", "Alba", "-item", "Bronze Armour", path)

	if got.code != ExitOK || !strings.Contains(got.stdout, "already has the design Bronze Armour") || !strings.Contains(got.stdout, "Nothing written") {
		t.Errorf("second add: exit code = %d, stdout = %q", got.code, got.stdout)
	}
	after, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(entries) || !bytes.Equal(mustRead(t, path), edited) {
		t.Error("the second add rewrote the save or made another backup")
	}
}

func TestAddItemRefuses(t *testing.T) {
	tests := []struct {
		name     string
		args     []string // the save path is appended
		wantCode int
		want     string
	}{
		{name: "missing item", args: []string{"-slot", "1", "-char", "Alba"}, wantCode: ExitUsage, want: "needs -item"},
		{name: "equipment by name", args: []string{"-slot", "1", "-char", "Alba", "-item", "Copper Sword"}, wantCode: ExitUsage, want: "no item that can be added is named"},
		{name: "equipment by ID", args: []string{"-slot", "1", "-char", "Alba", "-item", "1"}, wantCode: ExitUsage, want: "item ID 1 cannot be added; the allowed IDs are 293, 294-351, 381-392, 401-493"},
		{name: "magicite by ID", args: []string{"-slot", "1", "-char", "Alba", "-item", "256"}, wantCode: ExitUsage, want: "item ID 256 cannot be added"},
		{name: "artifact by ID", args: []string{"-slot", "1", "-char", "Alba", "-item", "231"}, wantCode: ExitUsage, want: "item ID 231 cannot be added"},
		{name: "seed by name", args: []string{"-slot", "1", "-char", "Alba", "-item", "Wheat Seed"}, wantCode: ExitUsage, want: "no item that can be added is named"},
		{name: "story item by ID", args: []string{"-slot", "1", "-char", "Alba", "-item", "365"}, wantCode: ExitUsage, want: "item ID 365 cannot be added"},
		{name: "placeholder by ID", args: []string{"-slot", "1", "-char", "Alba", "-item", "494"}, wantCode: ExitUsage, want: "item ID 494 cannot be added"},
		{name: "ambiguous name", args: []string{"-slot", "1", "-char", "Alba", "-item", "shard"}, wantCode: ExitUsage, want: "be more specific"},
		{name: "count of zero", args: []string{"-slot", "1", "-char", "Alba", "-item", "Bronze", "-count", "0"}, wantCode: ExitUsage, want: "-count must be between 1 and 64, got 0"},
		{name: "count above the inventory size", args: []string{"-slot", "1", "-char", "Alba", "-item", "Bronze", "-count", "65"}, wantCode: ExitUsage, want: "-count must be between 1 and 64, got 65"},
		{name: "more than the free slots", args: []string{"-slot", "1", "-char", "Alba", "-item", "Bronze", "-count", "62"}, wantCode: ExitError, want: "Alba has 61 free inventory slots of 64, not enough for 62"},
		{name: "design for another tribe", args: []string{"-slot", "1", "-char", "Alba", "-item", "Lunar Wpn."}, wantCode: ExitError, want: "Lunar Wpn. is a design for Selkies and Alba is a Clavat"},
		{name: "design for women", args: []string{"-slot", "1", "-char", "Alba", "-item", "471"}, wantCode: ExitError, want: "designs for men or for women are not supported"},
		{name: "two copies of a design", args: []string{"-slot", "1", "-char", "Alba", "-item", "Novice's Wpn.", "-count", "2"}, wantCode: ExitError, want: "can be added only once, not 2 times"},
		{name: "empty slot", args: []string{"-slot", "2", "-char", "Alba", "-item", "Bronze"}, wantCode: ExitError, want: "slot is empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, original := writeSample(t)

			got := run(append(append([]string{"add-item"}, tt.args...), path)...)

			if got.code != tt.wantCode || !strings.Contains(got.stderr, tt.want) {
				t.Errorf("exit code = %d, stderr = %q, want code %d and %q", got.code, got.stderr, tt.wantCode, tt.want)
			}
			entries, err := os.ReadDir(filepath.Dir(path))
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 || !bytes.Equal(mustRead(t, path), original) {
				t.Error("a refused edit left a changed save or extra files behind")
			}
		})
	}
}

func TestAddItemFillsTheInventoryExactly(t *testing.T) {
	path, _ := writeSample(t)

	// Alba holds 3 items, so 61 slots are free.
	got := run("add-item", "-slot", "1", "-char", "Alba", "-item", "Bronze", "-count", "61", path)
	if got.code != ExitOK || !strings.Contains(got.stdout, "items 3 -> 64 of 64") {
		t.Fatalf("exit code = %d, stdout = %q, stderr = %q", got.code, got.stdout, got.stderr)
	}
	if held := inventoryOf(t, path, 0, 0); len(held) != save.InventorySize {
		t.Errorf("%d items held, want %d", len(held), save.InventorySize)
	}

	full := run("add-item", "-slot", "1", "-char", "Alba", "-item", "Bronze", path)
	if full.code != ExitError || !strings.Contains(full.stderr, "0 free inventory slots") {
		t.Errorf("adding to a full inventory: exit code = %d, stderr = %q", full.code, full.stderr)
	}
}

// TestHelpCoversEveryCommand checks the single description of each
// command: it must be complete, its text must fit the terminal, and its
// examples must be real command lines of that command.
func TestHelpCoversEveryCommand(t *testing.T) {
	all := append(append([]*command{}, commands...), builtins...)
	overviewText := overview()
	seen := make(map[string]bool)

	for _, c := range all {
		if seen[c.name] {
			t.Errorf("command %q is listed twice", c.name)
		}
		seen[c.name] = true

		if c.summary == "" || strings.HasSuffix(c.summary, ".") {
			t.Errorf("%s: summary %q must be set and have no full stop", c.name, c.summary)
		}
		if len(c.examples) == 0 {
			t.Errorf("%s: has no example", c.name)
		}
		for _, ex := range c.examples {
			if !strings.HasPrefix(ex.line, "ffcc-gci "+c.name) || ex.note == "" {
				t.Errorf("%s: example %q must be a call of the command with a note", c.name, ex.line)
			}
		}
		if !strings.Contains(overviewText, "  "+c.name+" ") {
			t.Errorf("%s: missing from the overview", c.name)
		}

		help := c.help()
		for _, f := range c.allFlags() {
			if !strings.Contains(help, "  "+flagForm(f)+"\n") {
				t.Errorf("%s: help lacks the flag %q", c.name, flagForm(f))
			}
		}
		for _, line := range strings.Split(help, "\n") {
			// Usage and example lines are command lines and are not wrapped.
			if len(line) > helpWidth && !strings.HasPrefix(line, "  ffcc-gci ") {
				t.Errorf("%s: help line is %d characters wide: %q", c.name, len(line), line)
			}
		}
	}

	for _, c := range commands {
		if c.run == nil {
			t.Errorf("%s: has no code to run", c.name)
		}
	}
}

// TestExamplesAreAccepted runs every example of the commands that change
// or read a save against the sample save, so the documentation cannot
// show a command line the program refuses.
func TestExamplesAreAccepted(t *testing.T) {
	for _, c := range commands {
		for _, ex := range c.examples {
			t.Run(ex.line, func(t *testing.T) {
				if c.name == "menu" {
					useScript(t, ctrlC) // the menu opens and the user leaves it
				}
				dir := t.TempDir()
				// The sample has Alba in slot 1; the examples use slot 3.
				content := savetest.GCI(nil, nil, &savetest.Slot{
					Town: "Tipa", Key: 5,
					Characters: []savetest.Character{{Name: "Alba", Tribe: 0, Gil: 1}},
				})
				if err := os.WriteFile(filepath.Join(dir, "save.gci"), content, 0o644); err != nil {
					t.Fatal(err)
				}

				args := splitCommandLine(ex.line)[1:]
				for i, arg := range args {
					if arg == "save.gci" || arg == "edited.gci" {
						args[i] = filepath.Join(dir, arg)
					}
				}
				if got := run(args...); got.code != ExitOK {
					t.Errorf("exit code = %d, stderr = %q", got.code, got.stderr)
				}
			})
		}
	}
}

// splitCommandLine splits an example into arguments, honouring double
// quotes.
func splitCommandLine(line string) []string {
	var args []string
	var current strings.Builder
	quoted, pending := false, false
	for _, r := range line {
		switch {
		case r == '"':
			quoted = !quoted
			pending = true
		case r == ' ' && !quoted:
			if pending {
				args = append(args, current.String())
				current.Reset()
				pending = false
			}
		default:
			current.WriteRune(r)
			pending = true
		}
	}
	if pending {
		args = append(args, current.String())
	}
	return args
}

// TestUsageDocIsUpToDate fails when docs/USAGE.md no longer matches the
// reference the program renders. Regenerate it with "just docs".
func TestUsageDocIsUpToDate(t *testing.T) {
	const path = "../../docs/USAGE.md"
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v; create it with \"just docs\"", err)
	}
	if string(got) != usageMarkdown() {
		t.Errorf("%s is out of date; run \"just docs\" and commit the result", path)
	}

	viaCommand := run("help", "-markdown")
	if viaCommand.code != ExitOK || viaCommand.stdout != usageMarkdown() {
		t.Errorf("\"help -markdown\" does not print the reference: exit code %d", viaCommand.code)
	}
}

func TestHelpAll(t *testing.T) {
	got := run("help", "-all")

	if got.code != ExitOK {
		t.Fatalf("exit code = %d, stderr = %q", got.code, got.stderr)
	}
	for _, c := range append(append([]*command{}, commands...), builtins...) {
		if !strings.Contains(got.stdout, c.help()) {
			t.Errorf("the full help lacks the help of %q", c.name)
		}
	}
}

func TestWrap(t *testing.T) {
	tests := []struct {
		text   string
		width  int
		indent string
		want   string
	}{
		{text: "one two three", width: 20, want: "one two three"},
		{text: "one two three", width: 7, want: "one two\nthree"},
		{text: "one two three", width: 9, indent: "  ", want: "  one two\n  three"},
		{text: "  spaced   out  ", width: 20, want: "spaced out"},
		{text: "unbreakablyLongWord next", width: 5, want: "unbreakablyLongWord\nnext"},
		{text: "", width: 10, indent: "> ", want: "> "},
	}
	for _, tt := range tests {
		if got := wrap(tt.text, tt.width, tt.indent); got != tt.want {
			t.Errorf("wrap(%q, %d, %q) = %q, want %q", tt.text, tt.width, tt.indent, got, tt.want)
		}
	}
}
