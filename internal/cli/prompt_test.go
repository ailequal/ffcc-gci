package cli

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Keys as a terminal sends them.
const (
	keyEnter     = "\r"
	keyEsc       = "\x1b"
	keyCtrlC     = "\x03"
	keyDown      = "\x1b[B"
	keyLeft      = "\x1b[D"
	keyBackspace = "\x7f"
)

// typing returns a terminal whose keyboard is the given key presses, sent
// one after the other the way a person types them. What the questions
// draw is discarded.
func typing(t *testing.T, keys ...string) *terminal {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = w.Close()
		_ = r.Close()
	})
	go func() {
		for _, k := range keys {
			time.Sleep(15 * time.Millisecond)
			if _, err := io.WriteString(w, k); err != nil {
				return // the test is over
			}
		}
	}()
	return &terminal{in: r, out: io.Discard}
}

// answer runs one question and fails the test if it does not end in time,
// which is what a lost key press looks like.
func answer[T any](t *testing.T, ask func() (T, error)) (T, error) {
	t.Helper()
	type result struct {
		value T
		err   error
	}
	done := make(chan result, 1)
	go func() {
		value, err := ask()
		done <- result{value, err}
	}()
	select {
	case r := <-done:
		return r.value, r.err
	case <-time.After(10 * time.Second):
		t.Fatal("the question was not answered by the keys typed")
		panic("unreachable")
	}
}

var listOptions = []string{
	"159  Shuriken         Strength +1",
	"228  Earth Pendant    Gain another heart",
	"231  Sun Pendant      Gain another heart",
	"296  Mythril          Material for equipment",
}

func TestTerminalChoose(t *testing.T) {
	tests := []struct {
		name    string
		keys    []string
		want    int
		wantErr error
	}{
		{name: "Enter picks the first option", keys: []string{keyEnter}, want: 0},
		{name: "arrows move", keys: []string{keyDown, keyDown, keyEnter}, want: 2},
		{name: "the list wraps around", keys: []string{keyDown, keyDown, keyDown, keyDown, keyEnter}, want: 0},
		{name: "Esc goes back", keys: []string{keyDown, keyEsc}, wantErr: errBack},
		{name: "Ctrl+C quits", keys: []string{keyDown, keyCtrlC}, wantErr: errQuit},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			term := typing(t, tt.keys...)
			got, err := answer(t, func() (int, error) { return term.choose("Which?", listOptions) })
			if got != tt.want || !errors.Is(err, tt.wantErr) {
				t.Errorf("choose = %d, %v, want %d, %v", got, err, tt.want, tt.wantErr)
			}
		})
	}
}

func TestTerminalSearch(t *testing.T) {
	tests := []struct {
		name    string
		keys    []string
		want    int
		wantErr error
	}{
		{name: "Enter picks the first option", keys: []string{keyEnter}, want: 0},
		{name: "typing narrows the list", keys: []string{"s", "u", "n", keyEnter}, want: 2},
		{name: "typed all at once", keys: []string{"mythril", keyEnter}, want: 3},
		{name: "upper and lower case are the same", keys: []string{"PEND", keyEnter}, want: 1},
		{name: "arrows move in the narrowed list", keys: []string{"pend", keyDown, keyEnter}, want: 2},
		{name: "letters that move a plain list are text here", keys: []string{"k", "j", keyBackspace, keyBackspace, "g", keyEnter}, want: 0},
		{name: "deleting widens the list again", keys: []string{"sunx", keyBackspace, keyEnter}, want: 2},
		{name: "Enter does nothing when nothing matches", keys: []string{"zzz", keyEnter, keyEnter, keyEsc}, wantErr: errBack},
		{name: "Esc goes back", keys: []string{"sun", keyEsc}, wantErr: errBack},
		{name: "Ctrl+C quits", keys: []string{"sun", keyCtrlC}, wantErr: errQuit},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			term := typing(t, tt.keys...)
			got, err := answer(t, func() (int, error) { return term.search("Which?", listOptions) })
			if got != tt.want || !errors.Is(err, tt.wantErr) {
				t.Errorf("search = %d, %v, want %d, %v", got, err, tt.want, tt.wantErr)
			}
		})
	}
}

func TestTerminalInput(t *testing.T) {
	// The check accepts only "5000" and "1".
	check := func(text string) error {
		if text != "5000" && text != "1" {
			return errors.New("refused: " + text)
		}
		return nil
	}
	tests := []struct {
		name        string
		start       string
		keys        []string
		want        string
		wantErr     error
		wantRefused []string
	}{
		{name: "typed", keys: []string{"5000", keyEnter}, want: "5000"},
		{name: "Enter accepts what the field starts with", start: "1", keys: []string{keyEnter}, want: "1"},
		{name: "typing adds to what the field starts with", start: "500", keys: []string{"0", keyEnter}, want: "5000"},
		{
			name: "a refused answer is asked for again",
			keys: []string{"12", keyEnter, keyBackspace, keyBackspace, keyEnter, "5000", keyEnter},
			want: "5000", wantRefused: []string{"12", ""},
		},
		{name: "Esc goes back", keys: []string{"5000", keyEsc}, wantErr: errBack},
		{name: "Esc goes back after a refused answer", keys: []string{"12", keyEnter, keyEsc}, wantErr: errBack, wantRefused: []string{"12"}},
		{name: "Ctrl+C quits", keys: []string{"5000", keyCtrlC}, wantErr: errQuit},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var refused []string
			term := typing(t, tt.keys...)
			got, err := answer(t, func() (string, error) {
				return term.input("Gil", tt.start, func(text string) error {
					err := check(text)
					// The field also checks while it is being typed in;
					// what matters is what was refused at Enter.
					if err != nil && (len(refused) == 0 || refused[len(refused)-1] != text) {
						refused = append(refused, text)
					}
					return err
				})
			})
			if got != tt.want || !errors.Is(err, tt.wantErr) {
				t.Errorf("input = %q, %v, want %q, %v", got, err, tt.want, tt.wantErr)
			}
			for _, want := range tt.wantRefused {
				if !strings.Contains(strings.Join(refused, "|")+"|", want+"|") {
					t.Errorf("refused = %q, want it to include %q", refused, want)
				}
			}
		})
	}
}

func TestTerminalConfirm(t *testing.T) {
	tests := []struct {
		name    string
		keys    []string
		want    bool
		wantErr error
	}{
		{name: "Enter alone answers no", keys: []string{keyEnter}, want: false},
		{name: "y answers yes", keys: []string{"y"}, want: true},
		{name: "n answers no", keys: []string{"n"}, want: false},
		{name: "moving to Yes and Enter", keys: []string{keyLeft, keyEnter}, want: true},
		{name: "Esc goes back", keys: []string{keyEsc}, wantErr: errBack},
		{name: "Ctrl+C quits", keys: []string{keyCtrlC}, wantErr: errQuit},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			term := typing(t, tt.keys...)
			got, err := answer(t, func() (bool, error) { return term.confirm("Write?") })
			if got != tt.want || !errors.Is(err, tt.wantErr) {
				t.Errorf("confirm = %v, %v, want %v, %v", got, err, tt.want, tt.wantErr)
			}
		})
	}
}

func TestTerminalFile(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a-notes.txt", "save.gci", "save.gci.20261115-143005.bak"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("a .gci file is picked", func(t *testing.T) {
		term := typing(t, keyDown, keyEnter)
		got, err := answer(t, func() (string, error) { return term.file(dir) })
		if want := filepath.Join(dir, "save.gci"); got != want || err != nil {
			t.Errorf("file = %q, %v, want %q", got, err, want)
		}
	})
	t.Run("another file is not", func(t *testing.T) {
		term := typing(t, keyEnter, keyEnter, keyEsc)
		got, err := answer(t, func() (string, error) { return term.file(dir) })
		if got != "" || !errors.Is(err, errBack) {
			t.Errorf("file = %q, %v, want nothing picked and %v", got, err, errBack)
		}
	})
	t.Run("Ctrl+C quits", func(t *testing.T) {
		term := typing(t, keyCtrlC)
		if _, err := answer(t, func() (string, error) { return term.file(dir) }); !errors.Is(err, errQuit) {
			t.Errorf("file error = %v, want %v", err, errQuit)
		}
	})
}

func TestTerminalFrame(t *testing.T) {
	var out bytes.Buffer
	(&terminal{}).frame(&out, "ffcc-gci menu\nsecond line")

	want := "" +
		"╭───────────────╮\n" +
		"│ ffcc-gci menu │\n" +
		"│ second line   │\n" +
		"╰───────────────╯\n"
	// A buffer is not a terminal, so the frame comes without colours.
	if out.String() != want {
		t.Errorf("frame wrote:\n%s\nwant:\n%s", out.String(), want)
	}
}
