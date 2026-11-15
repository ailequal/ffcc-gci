package cli

import (
	"fmt"
	"io"

	"github.com/ailequal/ffcc-gci/internal/save"
)

// command is one subcommand: what it does, how it is called and the code
// that runs it. Everything shown to the user about a command comes from
// here.
type command struct {
	name     string
	summary  string    // one line without a full stop, for the command list
	synopsis string    // the arguments, as written after the command name
	flags    []flagDoc // the command's own flags
	edits    bool      // changes a save: also takes -slot, -char and -o
	notes    []string  // paragraphs of further explanation
	examples []example
	run      func(c *command, args []string, stdout, stderr io.Writer) error
}

// flagDoc documents one flag.
type flagDoc struct {
	name  string // without the dash
	value string // placeholder for the value, such as "<amount>"
	text  string
}

// example is one command line with a note on what it does.
type example struct {
	note string
	line string
}

// targetFlagDocs and outputFlagDoc are the flags shared by every command
// that changes a save: which character to edit, and where to write.
var (
	targetFlagDocs = []flagDoc{
		{"slot", "<1-4>", `Save slot to edit, as numbered by "ffcc-gci info".`},
		{"char", "<name or 1-8>", `Character to edit: the name, or the number "ffcc-gci info" shows.`},
	}
	outputFlagDoc = flagDoc{"o", "<new file>", "Write the result to this new file and leave the save untouched."}
)

// editNotes are the notes shared by every command that changes a save.
var editNotes = []string{
	"Without -o the save is edited in place, and the original is first copied to a backup " +
		"next to it, named <save.gci>.<date-time>.bak.",
	"Close Dolphin before editing: it rewrites the save while a game runs.",
}

// commands lists the subcommands in the order help shows them.
var commands = []*command{
	{
		name:     "info",
		summary:  "Show the slots and characters of a save file",
		synopsis: "<save.gci>",
		notes: []string{
			"Nothing is modified. For each of the four save slots it prints the town, the play " +
				"time and every character with gil, hearts, stats, items and artifacts. The " +
				"numbers it prints for slots and characters are the ones -slot and -char take.",
		},
		examples: []example{
			{"Look at a save", "ffcc-gci info save.gci"},
		},
		run: runInfo,
	},
	{
		name:     "set-gil",
		summary:  "Set a character's gil",
		synopsis: "-slot <1-4> -char <name or 1-8> -gil <amount> [-o <new file>] <save.gci>",
		flags: []flagDoc{
			{"gil", "<amount>", fmt.Sprintf(
				"New amount of gil, %d to %d. This is the range the game itself keeps gil in; "+
					"anything outside it is refused.", save.GilLimit.Min, save.GilLimit.Max)},
		},
		edits: true,
		examples: []example{
			{"Give Alba, in slot 3, 50000 gil", "ffcc-gci set-gil -slot 3 -char Alba -gil 50000 save.gci"},
			{"Pick the character by number instead of by name", "ffcc-gci set-gil -slot 3 -char 1 -gil 50000 save.gci"},
			{"Write a new file and leave the save untouched", "ffcc-gci set-gil -slot 3 -char Alba -gil 50000 -o edited.gci save.gci"},
		},
		run: runSetGil,
	},
	{
		name:     "add-artifact",
		summary:  "Grant a character an artifact",
		synopsis: "-slot <1-4> -char <name or 1-8> -artifact <name> [-o <new file>] <save.gci>",
		flags: []flagDoc{
			{"artifact", "<name>", fmt.Sprintf(
				"The artifact to grant: its name, a part of the name that matches only one "+
					"artifact, or its item ID (%d to %d). \"ffcc-gci artifacts\" lists them all.",
				save.ArtifactLimit.Min, save.ArtifactLimit.Max)},
		},
		edits: true,
		notes: []string{
			"An artifact is either owned or not, so it cannot be granted twice: if the " +
				"character already has it, nothing is written.",
			"The game works out strength, magic, defence, hearts and command slots from the " +
				"artifacts by itself, so the new values are there as soon as the save is loaded. " +
				"\"ffcc-gci info\" reads the numbers the game last stored, and keeps showing the " +
				"old ones until the game has saved again.",
		},
		examples: []example{
			{"Grant an artifact by name", `ffcc-gci add-artifact -slot 3 -char Alba -artifact "Sun Pendant" save.gci`},
			{"A part of the name is enough when it matches only one artifact", "ffcc-gci add-artifact -slot 3 -char Alba -artifact kaiser save.gci"},
			{"Grant an artifact by item ID", "ffcc-gci add-artifact -slot 3 -char Alba -artifact 231 save.gci"},
		},
		run: runAddArtifact,
	},
	{
		name:     "artifacts",
		summary:  "List the artifacts that can be granted",
		synopsis: "[text]",
		notes: []string{
			"The artifacts are grouped by what they improve, each with its item ID and the " +
				"game's description. With text, only artifacts whose name contains it are listed.",
		},
		examples: []example{
			{"List every artifact", "ffcc-gci artifacts"},
			{"List the artifacts with \"ring\" in the name", "ffcc-gci artifacts ring"},
		},
		run: runArtifacts,
	},
	{
		name:     "add-item",
		summary:  "Add an item to a character's inventory",
		synopsis: "-slot <1-4> -char <name or 1-8> -item <name> [-count <n>] [-o <new file>] <save.gci>",
		flags: []flagDoc{
			{"item", "<name>",
				"The item to add: its name, a part of the name that matches only one item, or " +
					"its item ID. \"ffcc-gci items\" lists what can be added; anything else is refused."},
			{"count", "<n>", fmt.Sprintf(
				"How many to add, 1 by default. Items do not stack: each one takes one of the "+
					"character's %d inventory slots, and the command refuses to add more than "+
					"there are free slots.", save.InventorySize)},
		},
		edits: true,
		notes: []string{
			"Designs have two extra rules. A design is added only once, and only if it is for " +
				"anyone or for the character's own tribe. Designs for men or for women are not " +
				"supported.",
		},
		examples: []example{
			{"Add one item", "ffcc-gci add-item -slot 3 -char Alba -item Mythril save.gci"},
			{"Add several of the same item", `ffcc-gci add-item -slot 3 -char Alba -item "Striped Apple" -count 5 save.gci`},
			{"Add a design", `ffcc-gci add-item -slot 3 -char Alba -item "Iron Armour" save.gci`},
			{"Add an item by item ID", "ffcc-gci add-item -slot 3 -char Alba -item 293 save.gci"},
		},
		run: runAddItem,
	},
	{
		name:     "items",
		summary:  "List the items that can be added",
		synopsis: "[text]",
		notes: []string{
			"The items are grouped as Phoenix Down, materials, food and designs, each with its " +
				"item ID. With text, only items whose name contains it are listed.",
			"Equipment, magicite, seeds, story items and unused entries of the game's item " +
				"table are left out for safety and cannot be added.",
		},
		examples: []example{
			{"List every item that can be added", "ffcc-gci items"},
			{"List the items with \"armour\" in the name", "ffcc-gci items armour"},
		},
		run: runItems,
	},
	{
		name:    "limits",
		summary: "Show every editable value and its allowed range",
		notes: []string{
			"Each value the program can set has a fixed range taken from the game's own code. " +
				"This prints every range and the reason it is safe.",
		},
		examples: []example{
			{"Show the ranges", "ffcc-gci limits"},
		},
		run: runLimits,
	},
}

// builtins are the commands handled by Run itself. They are listed after
// the others and documented the same way.
var builtins = []*command{
	{
		name:     "help",
		summary:  "Show help for the program or for one command",
		synopsis: "[command | -all | -markdown]",
		flags: []flagDoc{
			{"all", "", "Print the help of every command, one after the other."},
			{"markdown", "", "Print the full reference in Markdown. docs/USAGE.md is this output."},
		},
		notes: []string{
			"Without arguments it lists the commands. \"ffcc-gci <command> -h\" is the same as " +
				"\"ffcc-gci help <command>\".",
		},
		examples: []example{
			{"List the commands", "ffcc-gci help"},
			{"Show the flags and examples of one command", "ffcc-gci help add-item"},
			{"Show everything at once", "ffcc-gci help -all"},
		},
	},
	{
		name:    "version",
		summary: "Print the program version",
		examples: []example{
			{"Print the version", "ffcc-gci version"},
		},
	},
}

// allFlags returns the flags the command takes, in the order of its
// synopsis.
func (c *command) allFlags() []flagDoc {
	if !c.edits {
		return c.flags
	}
	flags := append([]flagDoc{}, targetFlagDocs...)
	flags = append(flags, c.flags...)
	return append(flags, outputFlagDoc)
}

// allNotes returns the notes of the command, its own first.
func (c *command) allNotes() []string {
	if !c.edits {
		return c.notes
	}
	return append(append([]string{}, c.notes...), editNotes...)
}

// usageLine returns the command line that calls the command.
func (c *command) usageLine() string {
	line := "ffcc-gci " + c.name
	if c.synopsis != "" {
		line += " " + c.synopsis
	}
	return line
}
