package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/term"
)

// Errors a question returns when the user leaves it without answering.
var (
	errBack = errors.New("the question was left with Esc")    // go back one step
	errQuit = errors.New("the question was left with Ctrl+C") // leave the menu
)

// prompter asks the user questions in the terminal. The menu talks to the
// user only through it, so the tests can script the answers.
//
// Every question returns errBack when the user presses Esc and errQuit
// when the user presses Ctrl+C.
type prompter interface {
	// choose shows a short list and returns the index of the option the
	// user picked.
	choose(header string, options []string) (int, error)
	// search is choose for a long list: typing narrows it down.
	search(header string, options []string) (int, error)
	// input asks for one line of text. The field starts with value in it,
	// and an answer check refuses is asked for again, with the reason.
	input(header, value string, check func(string) error) (string, error)
	// confirm asks a question answered with yes or no. No is the default.
	confirm(question string) (bool, error)
	// file lets the user pick a save file, starting in the directory dir.
	file(dir string) (string, error)
	// frame writes text to w, set apart from the output around it.
	frame(w io.Writer, text string)
}

// newPrompter returns the prompter the menu uses. Tests replace it.
var newPrompter = func() (prompter, error) {
	if !term.IsTerminal(os.Stdin.Fd()) || !term.IsTerminal(os.Stdout.Fd()) {
		return nil, errors.New("the menu asks its questions in a terminal, and this is not one; " +
			"the other commands work anywhere: run \"ffcc-gci help\"")
	}
	return &terminal{}, nil
}

// terminal asks its questions with the forms of the huh library, which
// are drawn in the terminal and answered with the keyboard.
type terminal struct {
	in  io.Reader // where key presses are read from; nil for the keyboard
	out io.Writer // where the questions are drawn; nil for the terminal
}

// listHeight is the number of lines a long list takes, with its title.
const listHeight = 16

// question is one form, run as a program of its own. It ends when the
// form has its answer, or when the user leaves it: with Esc to go back
// one step, with Ctrl+C to leave the menu.
//
// The form is not run with its own Run method, because that reports both
// keys the same way and leaves a line of the form on the screen.
type question struct {
	form *huh.Form
	left error // errBack or errQuit, once the user has left
}

func (q *question) Init() tea.Cmd { return q.form.Init() }

func (q *question) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if pressed, ok := msg.(tea.KeyPressMsg); ok {
		switch pressed.String() {
		case "esc":
			q.left = errBack
			return q, tea.Quit
		case "ctrl+c":
			q.left = errQuit
			return q, tea.Quit
		}
	}
	model, cmd := q.form.Update(msg)
	if form, ok := model.(*huh.Form); ok {
		q.form = form
	}
	return q, cmd
}

// View draws the form. Once it is over it draws nothing, so that the
// question is erased from the screen.
func (q *question) View() tea.View {
	if q.left != nil {
		return tea.NewView("")
	}
	return tea.NewView(q.form.View())
}

// ask shows one field and waits for the answer.
//
// Every question is a program that takes over the keyboard and gives it
// back. Bubble Tea before v2.0.9 gave it back in a state in which the
// next question lost key presses; the tests that run the menu in a real
// terminal (menu_e2e_test.go) fail when that comes back.
func (t *terminal) ask(field huh.Field, keys *huh.KeyMap) error {
	form := huh.NewForm(huh.NewGroup(field)).WithKeyMap(keys)
	form.SubmitCmd = tea.Quit // an answered form ends the program
	form.CancelCmd = tea.Quit

	var options []tea.ProgramOption
	if t.in != nil {
		options = append(options, tea.WithInput(t.in))
	}
	if t.out != nil {
		options = append(options, tea.WithOutput(t.out))
	}
	q := &question{form: form}
	_, err := tea.NewProgram(q, options...).Run()

	switch {
	case q.left != nil:
		return q.left
	case errors.Is(err, tea.ErrInterrupted):
		return errQuit
	case err != nil:
		return fmt.Errorf("could not ask in the terminal: %w", err)
	case q.form.State != huh.StateCompleted:
		return errors.New("the question ended without an answer")
	}
	return nil
}

// list shows options and returns the index of the one picked.
//
// With search the list keeps its filter open, so that typing narrows it
// down at once. The filter takes the place of the title, so the header
// is shown as the description below it.
func (t *terminal) list(header string, options []string, search bool) (int, error) {
	choices := make([]huh.Option[int], len(options))
	for i, option := range options {
		choices[i] = huh.NewOption(option, i)
	}
	picked := 0
	field := huh.NewSelect[int]().Title(header).Options(choices...).Value(&picked)
	keys := huh.NewDefaultKeyMap()
	if search {
		field = field.Description(header + "\nType a part of the name to narrow the list down.").
			Height(listHeight).Filtering(true)
		// A list that starts with its filter open shows the help of every
		// key, also of those that do nothing here.
		keys.Select.Filter.SetEnabled(false)
		keys.Select.Prev.SetEnabled(false)
		keys.Select.Next.SetEnabled(false)
	}
	if err := t.ask(field, keys); err != nil {
		return 0, err
	}
	return picked, nil
}

func (t *terminal) choose(header string, options []string) (int, error) {
	return t.list(header, options, false)
}

func (t *terminal) search(header string, options []string) (int, error) {
	return t.list(header, options, true)
}

func (t *terminal) input(header, value string, check func(string) error) (string, error) {
	text := value
	if err := t.ask(huh.NewInput().Title(header).Value(&text).Validate(check), huh.NewDefaultKeyMap()); err != nil {
		return "", err
	}
	return text, nil
}

func (t *terminal) confirm(question string) (bool, error) {
	yes := false // "No" is selected, so that Enter alone never writes
	if err := t.ask(huh.NewConfirm().Title(question).Affirmative("Yes").Negative("No").Value(&yes), huh.NewDefaultKeyMap()); err != nil {
		return false, err
	}
	return yes, nil
}

func (t *terminal) file(dir string) (string, error) {
	path := ""
	// Hidden entries are shown: on Linux, Dolphin keeps its saves under
	// the hidden directory ~/.local.
	field := huh.NewFilePicker().Title("Pick the save file").CurrentDirectory(dir).
		AllowedTypes([]string{".gci"}).ShowHidden(true).Picking(true).Height(listHeight).Value(&path)
	if err := t.ask(field, huh.NewDefaultKeyMap()); err != nil {
		return "", err
	}
	return path, nil
}

// frame draws a border around text. Colours are left out by lipgloss
// when w is not a terminal that shows them.
func (t *terminal) frame(w io.Writer, text string) {
	style := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("212")).Padding(0, 1)
	_, _ = lipgloss.Fprintln(w, style.Render(text))
}
