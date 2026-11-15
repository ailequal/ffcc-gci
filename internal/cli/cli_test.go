package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ailequal/ffcc-gci/internal/gci"
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

// writeSample writes the sample save to a temporary directory and returns
// its path and contents.
func writeSample(t *testing.T) (path string, content []byte) {
	t.Helper()
	content = sampleGCI()
	path = filepath.Join(t.TempDir(), "save.gci")
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
		{name: "version", args: []string{"version"}, wantCode: ExitOK, wantStdout: "ffcc-gci " + Version},
		{name: "unknown command", args: []string{"bogus"}, wantCode: ExitUsage, wantStderr: `unknown command "bogus"`},
		{name: "command help", args: []string{"set-gil", "-h"}, wantCode: ExitOK, wantStderr: "Close Dolphin"},
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
		"Stats      strength 10, magic 11, defence 12",
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
	for _, l := range save.Limits() {
		want := fmt.Sprintf("%s: %d to %d", l.Name, l.Min, l.Max)
		if !strings.Contains(got.stdout, want) || !strings.Contains(got.stdout, l.Source) {
			t.Errorf("output lacks the %q limit or its source\n%s", l.Name, got.stdout)
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
	if got.code != ExitOK || !strings.Contains(got.stderr, want) {
		t.Errorf("exit code = %d, help = %q, want it to state %q", got.code, got.stderr, want)
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
