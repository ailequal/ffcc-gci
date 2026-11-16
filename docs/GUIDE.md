# ffcc-gci guide

This guide walks through the things you will actually do with ffcc-gci, step
by step: finding the save, looking at it, editing it, checking the result in
the game and undoing an edit.

For the complete list of commands, flags, artifacts and items, see the
[usage reference](USAGE.md).

The examples use a character named Alba in slot 3 of a file named `save.gci`.
Use your own slot, character and file. Every command line in this guide is run
by the project's tests, so the examples are ones the program accepts.

- [1. Find your save](#1-find-your-save)
- [2. Look at the save](#2-look-at-the-save)
- [3. Edit with the menu](#3-edit-with-the-menu)
- [4. Edit with commands](#4-edit-with-commands)
- [5. Check the result in the game](#5-check-the-result-in-the-game)
- [6. Undo an edit](#6-undo-an-edit)
- [7. When the program refuses](#7-when-the-program-refuses)
- [8. Names and languages](#8-names-and-languages)

## 1. Find your save

**Close Dolphin first**, or at least stop the game. While a game runs, Dolphin
keeps the save in memory and writes it back, which would undo your edit.

The easiest way to find the save is to let the menu do it. Without a file, it
looks in Dolphin's folders and lists the saves of the game it finds there:

```sh
ffcc-gci menu
```

To find it yourself, open Dolphin's user folder (in Dolphin: **File > Open
User Folder**) and go into `GC/EUR/Card A`. The save is a `.gci` file, usually
named `01-GCCP-FFCC.gci`. The user folder is normally here:

| System | Dolphin's user folder |
|---|---|
| Linux | `~/.local/share/dolphin-emu` |
| Linux, Flatpak | `~/.var/app/org.DolphinEmu.dolphin-emu/data/dolphin-emu` |
| macOS | `~/Library/Application Support/Dolphin` |
| Windows | `%APPDATA%\Dolphin Emulator`, or `Dolphin Emulator` in your Documents folder for older installations |

**The path has a space in it** (`Card A`), so put it in quotes:

```sh
ffcc-gci info "$HOME/.local/share/dolphin-emu/GC/EUR/Card A/01-GCCP-FFCC.gci"
```

On Windows, in PowerShell:

```powershell
ffcc-gci info "$env:APPDATA\Dolphin Emulator\GC\EUR\Card A\01-GCCP-FFCC.gci"
```

Or go into the folder once and use the short name from then on:

```sh
cd "$HOME/.local/share/dolphin-emu/GC/EUR/Card A"
ffcc-gci info 01-GCCP-FFCC.gci
```

**If there is no `.gci` file there**, Dolphin is keeping your saves in a single
memory card file (`.raw`), which ffcc-gci does not read. Open **Tools > Memory
Card Manager** in Dolphin, select the save and choose **Export as .gci**. To
play the edited save, set **Slot A** to **GCI Folder** in Dolphin's GameCube
settings and put the file in `GC/EUR/Card A`.

Only saves of the European (PAL) version of the game are supported. Any other
file is refused and left untouched.

## 2. Look at the save

`info` changes nothing. It shows the four save slots of the file and every
character in them:

```sh
ffcc-gci info save.gci
```

```text
save.gci: Final Fantasy Crystal Chronicles (PAL)

Slot 1: empty

Slot 2: empty

Slot 3: town Tipa, play time 1:15
  1. Alba (Clavat, in party)
     Gil        200
     Hearts     4 of 4
     Stats      strength 10, magic 11, defence 12
     Items      2 of 64: Phoenix Down, Mythril
     Artifacts  1: Sun Pendant
  2. Bruno (Selkie, in party)
     Gil        346
     ...

Slot 4: empty
```

What to read from it:

- **The slot number** (`Slot 3`) is what `-slot` takes. A slot is one saved
  game: a town and its caravan.
- **The character number or name** (`1` or `Alba`) is what `-char` takes. Gil,
  items and artifacts belong to a character, not to the whole slot.
- **Items** shows how many of the 64 inventory slots are used. Items do not
  stack: three Mythril take three slots.

## 3. Edit with the menu

The menu asks for everything, so there are no flags to remember:

```sh
ffcc-gci menu save.gci
```

1. **Pick the slot and the character.** A question with only one possible
   answer is skipped.
2. **Pick what to do**: set gil, add an artifact, add an item.
3. **Answer the question.**
   - For gil, type the amount. A value outside 0 to 99,999,999 is refused on
     the spot.
   - For an artifact or an item, pick it from the list. The list holds only
     what this character can receive. Type a part of a name to narrow it down;
     the item ID and the description are searched too, so `design` leaves only
     designs and `296` finds Mythril.
4. **Confirm.** The menu shows the change and the command that makes it. "No"
   is selected, so move to "Yes" or press `y` to write it.
5. **Carry on or quit.** You are back at step 2 for the same character.

| Key | What it does |
|---|---|
| Up, Down | Move in a list |
| Enter | Pick or confirm |
| Letters and digits | Narrow down a long list |
| Left, Right, `y`, `n` | Answer a confirmation |
| Esc | Go back one step |
| Ctrl+C | Quit the menu |

The menu makes exactly the edits the commands below make, with the same checks
and the same backup. It needs a terminal, so in a script use the commands.

## 4. Edit with commands

Every editing command takes the slot, the character and the save file. The
flags can come before or after the file.

### Gil

```sh
# Set Alba's gil to 50000
ffcc-gci set-gil -slot 3 -char Alba -gil 50000 save.gci

# The same, picking the character by the number info shows
ffcc-gci set-gil -slot 3 -char 1 -gil 50000 save.gci

# The most the game allows
ffcc-gci set-gil -slot 3 -char Alba -gil 99999999 save.gci
```

### Artifacts

Artifacts are how the game raises strength, magic, defence, hearts and command
slots. Look at what exists, then grant one:

```sh
# Every artifact, grouped by what it improves
ffcc-gci artifacts

# Only the ones with "pendant" in the name
ffcc-gci artifacts pendant

# Grant one by name; quote a name with a space in it
ffcc-gci add-artifact -slot 3 -char Alba -artifact "Moon Pendant" save.gci

# A part of the name is enough when it matches only one artifact
ffcc-gci add-artifact -slot 3 -char Alba -artifact kaiser save.gci

# Or use the item ID from the list
ffcc-gci add-artifact -slot 3 -char Alba -artifact 230 save.gci
```

An artifact is either owned or not. Granting one the character already has
changes nothing and says so.

### Items

Phoenix Down, materials, food and designs can be added: 164 items in all.

```sh
# Everything that can be added, grouped by kind
ffcc-gci items

# Only the items with "apple" in the name
ffcc-gci items apple

# Add one
ffcc-gci add-item -slot 3 -char Alba -item "Phoenix Down" save.gci

# Add five of the same; each one takes an inventory slot
ffcc-gci add-item -slot 3 -char Alba -item "Striped Apple" -count 5 save.gci

# A name that is also part of other names means the exact one: this adds Iron,
# not Iron Shard
ffcc-gci add-item -slot 3 -char Alba -item Iron save.gci
```

### Forging equipment

Weapons and armour cannot be added directly. Add what the blacksmith needs
instead: the design and the materials.

```sh
# Find the designs: each line says who can forge it
ffcc-gci items armour

# Add the design; a design is added once
ffcc-gci add-item -slot 3 -char Alba -item "Iron Armour" save.gci

# Add materials
ffcc-gci add-item -slot 3 -char Alba -item Iron -count 2 save.gci
ffcc-gci add-item -slot 3 -char Alba -item Mythril save.gci
```

In the game, the blacksmith then lists the design and shows which materials
and how much gil it asks for. A character can be given only designs marked
"for anyone" or for their own tribe.

### Several edits, or a copy

Commands can follow one another. Each one makes a backup of its own:

```sh
ffcc-gci set-gil -slot 3 -char Alba -gil 50000 save.gci
ffcc-gci add-artifact -slot 3 -char Alba -artifact "Star Pendant" save.gci
ffcc-gci add-item -slot 3 -char Alba -item Mythril -count 3 save.gci
```

To leave the save untouched and write the result to a new file, add `-o`. The
new file must not exist yet. Do not keep it in Dolphin's `Card A` folder next
to the original: Dolphin rejects two `.gci` files of the same save.

```sh
ffcc-gci set-gil -slot 3 -char Alba -gil 50000 -o edited.gci save.gci
```

## 5. Check the result in the game

Start Dolphin and load the save you edited.

| After | Look for |
|---|---|
| `set-gil` | The character's gil in the game shows the new amount. |
| `add-artifact` | The artifact is in the character's artifact list, and its bonus counts: a pendant adds a heart, for example. |
| `add-item` | The items are in the character's item list. An added design is offered by the blacksmith. |

After `add-artifact`, `ffcc-gci info` lists the new artifact but still prints
the old hearts and stats. That is expected: the game works those numbers out
by itself when it loads the save, and `info` reads the ones the game last
stored. They change in `info` once the game has saved again.

If the game shows the save as it was before, Dolphin was probably running
during the edit and wrote its own copy back. Close Dolphin and edit again.

## 6. Undo an edit

Every edit in place first copies the save to a backup next to it:

```text
save.gci
save.gci.20261115-143005.bak      the save before the first edit
save.gci.20261115-143210.bak      the save before the second edit
save.gci.20261115-143210-2.bak    a further edit within the same second
```

The date and time are in the name, so **the oldest backup is the save before
any edit**. To undo, close Dolphin and copy a backup back over the save:

```sh
cp save.gci.20261115-143005.bak save.gci
```

On Windows:

```powershell
Copy-Item save.gci.20261115-143005.bak save.gci
```

The backups are never deleted or overwritten by ffcc-gci. Delete them yourself
once you are happy with the save. Dolphin ignores them, because their names do
not end in `.gci`.

## 7. When the program refuses

A refusal never changes the save. These are the messages you are most likely
to meet:

| Message | What it means | What to do |
|---|---|---|
| `slot 1: slot is empty` | That slot of the file holds no save. | Run `info` and use a slot that lists characters. |
| `there is no character named "Carla"; available: 1 (Alba), 2 (Bruno)` | The slot has no character of that name. | Use one of the names or numbers the message lists. |
| `2 characters are named "Twin"; pick one by number` | Two characters share the name. | Use the number, as in `-char 2`. |
| `gil must be between 0 and 99999999` | The amount is outside what the game allows. | Use an amount in the range. `ffcc-gci limits` shows every range. |
| `"ring" matches 8 artifacts, be more specific` | The text is part of several names. | Type more of the name, or use the item ID. |
| `no item that can be added is named "Copper Sword"` | The item exists in the game but is not one the program adds, or the name is misspelt. | Run `ffcc-gci items` for the list. Equipment is obtained by [forging](#forging-equipment). |
| `item ID 1 cannot be added; the allowed IDs are 293, 294-351, 381-392, 401-493` | The ID is outside the items that are proven safe to add. | Pick an ID from `ffcc-gci items`. |
| `has 0 free inventory slots of 64, not enough for 1` | The character's inventory is full, or has fewer free slots than `-count`. | Add fewer, or make room in the game first. |
| `is a design for Clavats and Bruno is a Selkie, who could not forge it` | The design is for another tribe. | Give it to a character of that tribe, or pick a design "for anyone". |
| `designs for men or for women are not supported` | The program cannot yet tell a character's gender with certainty. | Pick another design. |
| `is a design and can be added only once` | `-count` was used with a design. | Leave `-count` out. |
| `changed on disk while it was being edited; is Dolphin running?` | Something else wrote the save during the edit. | Close Dolphin and run the command again. |
| `not a valid GCI file` or `unsupported save file` | The file is not a save of the PAL version of this game. | Check that you picked the right file: see [Find your save](#1-find-your-save). |
| `slot does not hold a valid save` | The slot fails the checks the game itself makes when loading, so the game would not load it either. | Use another slot, or restore a backup. |
| `could not write the new file: ... file exists` | The file given to `-o` already exists. | Give `-o` a new name. |
| `no such file or directory` | The path is wrong, often because of the space in `Card A`. | Put the path in quotes. |
| `the menu asks its questions in a terminal, and this is not one` | `menu` was started from a script or with its input redirected. | Run it in a terminal, or use the commands. |

These are not refusals; the command succeeds and writes nothing, because there
is nothing to do:

- `gil is already 200. Nothing written.`
- `already has Sun Pendant. An artifact cannot be held twice. Nothing written.`
- `already has the design Iron Armour. A design is added only once. Nothing written.`

The README lists [what the program will not do](../README.md#what-it-will-not-do),
and why.

## 8. Names and languages

The names of artifacts and items are the English ones of the PAL game. If you
play in another language, the game shows its own names for the same things.

The item IDs are the same in every language. To find an item whose English
name you do not know:

```sh
# Look through the lists: artifacts are grouped by what they improve, items by kind
ffcc-gci artifacts
ffcc-gci items

# Then use the item ID
ffcc-gci add-item -slot 3 -char Alba -item 296 save.gci
```

In the menu, the same lists can be searched by ID and by description.
