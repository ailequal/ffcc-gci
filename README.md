# ffcc-gci

A command-line save editor for **Final Fantasy Crystal Chronicles** on the
GameCube. It works on save files in Dolphin's `.gci` format.

> **PAL only.** Only the European version of the game (`GCCP01`) is supported.
> Saves from the USA and Japanese versions are refused.

It is built around one rule: **edit only what is proven safe**. An edit is
supported only when the game's own code, as recovered by the
[FFCC decompilation](https://github.com/zcanann/FFCC-Decomp), shows that the
game accepts it, and the result is a save the game could have produced by
itself. Everything else is refused.

> **AI usage disclosure**: This codebase was written with LLM-based tools. It is reviewed and tested, but bugs and odd patterns are possible. Review before relying on it in production.

## What it can do

| Edit | Command | Accepted values |
|---|---|---|
| Show a save: slots, characters, gil, stats, items, artifacts | `info` | Read-only |
| Set a character's gil | `set-gil` | 0 to 99,999,999 |
| Grant a character an artifact | `add-artifact` | The game's 73 artifacts |
| Add items to a character's inventory | `add-item` | Phoenix Down, materials, food and designs: 164 items, up to 64 per character |
| Make the same edits by answering questions | `menu` | The same values |

Each kind of edit was confirmed in the real game on a PAL save.

Strength, magic, defence and hearts cannot be set directly, because the game
recalculates them. Artifacts are the way to raise them, as in normal play.

## Install

**Download a release.** The
[releases page](https://github.com/ailequal/ffcc-gci/releases) has a ready-made
program for Linux, macOS and Windows. Unpack the archive for your system and
run `ffcc-gci help`. Each release lists how to verify the download.

**Or install it with Go.** You need [Go](https://go.dev/dl/) 1.27.4 or newer.
This makes `ffcc-gci` run from any folder:

```sh
go install github.com/ailequal/ffcc-gci/cmd/ffcc-gci@latest
```

**Or build it from a clone of this repository**, with
[just](https://just.systems) or with Go alone:

```sh
just build      # writes bin/ffcc-gci
just install    # or install it into Go's bin directory

go build -o bin/ffcc-gci ./cmd/ffcc-gci    # the same without just
```

## Quick start

Close Dolphin first, then:

```sh
# Look at the save: this shows the slot and character numbers
ffcc-gci info save.gci

# Give the character Alba in slot 3 some gil
ffcc-gci set-gil -slot 3 -char Alba -gil 50000 save.gci

# See which artifacts exist, then grant one
ffcc-gci artifacts
ffcc-gci add-artifact -slot 3 -char Alba -artifact "Sun Pendant" save.gci

# See which items can be added, then add two of one
ffcc-gci items
ffcc-gci add-item -slot 3 -char Alba -item Mythril -count 2 save.gci
```

## The menu

If you would rather not type flags, `ffcc-gci menu` asks for everything
instead:

```sh
ffcc-gci menu save.gci    # or just "ffcc-gci menu" to pick the save from a list
```

```text
╭───────────────────────────────────────╮
│ ffcc-gci menu                         │
│ Esc goes back one step, Ctrl+C quits. │
│ Close Dolphin before editing.         │
╰───────────────────────────────────────╯
Save: /home/you/.local/share/dolphin-emu/GC/EUR/Card A/01-GCCP-FFCC.gci

┃ Slot 3, Alba (Clavat): 200 gil, 10 of 64 items, 0 artifacts
┃ > Set gil
┃   Add an artifact
┃   Add an item
┃   Choose another character
┃   Show the whole save
┃   Show what can be edited
┃   Quit

↑ up • ↓ down • / filter • enter submit
```

- You pick the slot, the character and the change from lists. Long lists, such
  as the 164 items, narrow down as you type a part of a name.
- Only what the character can receive is listed: no artifact they already
  have, no design they could not forge.
- Without a save file, the menu offers the saves it finds in Dolphin's
  European memory card folders, or lets you browse for one.
- The menu changes nothing by itself. It shows the `set-gil`, `add-artifact`
  or `add-item` command that makes the edit, asks you to confirm, and runs
  that command. The checks and the backup are the same, and "No" is the
  default answer.

The menu is part of the program: there is nothing else to install. It needs a
terminal to run in, so in a script use the other commands.

## Guide and reference

**[docs/GUIDE.md](docs/GUIDE.md) is the step-by-step guide**: where Dolphin
keeps the save, how to edit it with the menu or with commands, what to look
for in the game afterwards, how to undo an edit, and what to do when the
program refuses.

**[docs/USAGE.md](docs/USAGE.md) is the full reference**: every command, flag
and example, and the lists of all artifacts and items. The program prints the
same information:

```sh
ffcc-gci help               # the list of commands
ffcc-gci help add-item      # the flags and examples of one command
ffcc-gci help -all          # everything at once
```

## Before you edit

- **Close Dolphin, or at least stop the game.** Dolphin keeps the save in
  memory while a game runs and writes it back, which would undo your edit.
- **Find the save.** It is a `.gci` file in Dolphin's GCI folder, usually
  `GC/EUR/Card A/` inside Dolphin's user directory; `ffcc-gci menu` looks
  there for you, and the [guide](docs/GUIDE.md#1-find-your-save) says where
  that directory is on each system. The path has a space in it, so quote it.
  Saves on a raw memory card image (`.raw`) are not supported; export them to
  `.gci` with Dolphin's memory card manager.
- **Know how to undo.** An edit first copies the original to
  `<save.gci>.<date-time>.bak` next to the save. To undo, copy that file back
  over the save. Every edit makes a backup of its own, so the oldest one is
  the save as it was before any edit. With `-o <new file>` the save is not
  touched at all.
- **Expect old numbers in `info` after granting an artifact.** The game counts
  the artifact as soon as it loads the save, but `info` reads the hearts and
  stats the game last stored, so they change only after the game has saved
  again.

## How it stays safe

- **Validation first.** The file must be a PAL save of this game with the
  expected size, and the slot must pass the checks the game applies on load
  (header tags, region, checksum). Otherwise nothing is edited.
- **Fixed ranges.** Each value has a range taken from the game's own code, and
  it is checked twice: when the command line is read and when the value is
  stored. `ffcc-gci limits` lists every range and the evidence behind it.
- **The original is kept.** An edit in place always makes a backup first. The
  backup does not end in `.gci` on purpose: Dolphin loads every `.gci` in a
  card folder and rejects duplicates. For the same reason, do not put a `-o`
  output next to the original in that folder.
- **Self-check before writing.** The edited file is decoded again and compared
  with the original. If anything other than the intended bytes and the
  checksum changed, nothing is written.
- **No half-written saves.** The new file is written beside the save and
  renamed over it in one step. If the save changed on disk in the meantime,
  the edit is abandoned.

## What it will not do

Safety comes first, so the tool refuses whatever it cannot prove safe.

| Not supported | Why |
|---|---|
| Equipment (weapons, armour, accessories) | It is tied to a tribe, and the game does not re-check what a character has equipped. |
| Magicite and command items | They only exist inside a dungeon. The game deletes them when the party leaves, so no real save holds them. |
| Seeds | They feed town events that have not been traced in the decompilation. |
| Story items (Worn Bandana, Shella Mark, Kilanda Sulphur, Cactus Flower) | The game's scripts check for them together with story progress. |
| Goddess Stat., Devil's Mask, Ultimite, Dark Sphere, Wheat, Flour and the elemental shards | Their role in the game's scripts has not been verified. |
| Unused entries of the item table ("Extra 22", "design 96", "Equip. 187" and so on) | The game has no description for them; they were never meant to be held. |
| Designs for men or for women | The tool cannot yet tell a character's gender with certainty. |
| A design for another tribe | The character could not forge it. |
| A second copy of a design | One copy is all a character needs; duplicates of designs were not verified. |
| Removing items | Equipment and command slots refer to inventory slots by position, so removing an item can leave them pointing at the wrong thing. |
| Setting strength, magic, defence or hearts directly | The game overwrites them with its own calculation. Grant artifacts instead. |
| Changing name, tribe, gender or appearance | They are tied to data the tool does not model, such as base stats and textures. |
| Creating characters or save slots | Too much state would have to be invented. |
| Story progress, event flags and letters | Their meaning has not been verified. |
| USA and Japanese saves | Only the PAL version has been checked and tested. |

## Documentation

| Document | Content |
|---|---|
| [docs/GUIDE.md](docs/GUIDE.md) | Step by step: finding the save, looking at it, editing with the menu or with commands, checking the result in the game, undoing an edit, and what each refusal means. |
| [docs/USAGE.md](docs/USAGE.md) | Every command, flag, example and accepted value, with the full lists of artifacts and items. |
| [docs/SAVE_FORMAT.md](docs/SAVE_FORMAT.md) | The save file format, what the game recalculates, and the evidence that makes each edit safe. |
| [CHANGELOG.md](CHANGELOG.md) | What changed in each version. |

## Getting help

Start with the [guide](docs/GUIDE.md), which explains the messages the program
prints when it refuses, then `ffcc-gci help <command>` and
[docs/USAGE.md](docs/USAGE.md). For a problem or a question, open an issue at
<https://github.com/ailequal/ffcc-gci/issues>. If the game rejects an edited
save, [restore a backup](docs/GUIDE.md#6-undo-an-edit) and include the output
of `ffcc-gci info` on the save in the issue.

## Development

The development tasks are recipes in the [`justfile`](justfile), run with
[just](https://just.systems) 1.55 or newer
([how to install it](https://just.systems/man/en/packages.html)). `just` alone
lists them by group:

```sh
just                    # list every recipe
just check              # format check, vet, tests and the changelog format; run it before every commit
just run menu save.gci  # build the program and run it with these arguments
just test-short         # the tests without the ones that need a terminal
just test-terminal 30   # only the terminal tests, 30 times over
just lint-docker        # golangci-lint, the pinned release in Docker, as the workflow runs it
just docs               # regenerate docs/USAGE.md after changing a command
just dist               # cross-compile the release archives into dist/
just release-status     # say what a push to main would release
```

The top of the `justfile` lists a few `just` options worth knowing, such as
`just --show <recipe>` to read a recipe and `just --dry-run <recipe>` to see
what it would run.

Each command is described once, in `internal/cli/commands.go`. The help the
program prints and `docs/USAGE.md` are both rendered from that description,
and a test fails when the document is out of date. Another test runs every
example in it against a sample save.

`docs/GUIDE.md` and this README are written by hand, and tested as well:
every `ffcc-gci` command line in them is run against a sample save, every
message the guide quotes is compared with what the program prints, and every
link between the documents is followed.

Everything that reads or writes a save uses only Go's standard library. The
menu is drawn with [huh](https://github.com/charmbracelet/huh), which is
compiled into the program.

The menu has three layers of tests. Its logic is tested with scripted answers,
and its questions with scripted key presses. The third layer runs the built
program in a real terminal, a [tmux](https://github.com/tmux/tmux) session,
and types into it; those tests are skipped when tmux is not installed, and
`just test-short` leaves them out.

The tests build synthetic saves in memory. Real saves are never committed:
they are personal data and embed the game's banner art. To also run the tests
against a real PAL save, put one at `testdata/pal.gci` (git-ignored).

| Path | Responsibility |
|---|---|
| `cmd/ffcc-gci` | Program entry point. |
| `internal/cli` | Commands, help, output, file writing and the menu. |
| `internal/gci` | The GCI container (header and blocks). |
| `internal/save` | Save slots: decoding, checksum, character fields, limits, edit verification. |
| `internal/save/savetest` | Synthetic saves for tests. |
| `internal/itemdb` | Item IDs, names and the game's own description of each item. |
| `docs` | The guide, the usage reference and the save format notes. |
| `scripts` | The release build and the changelog reader used by the workflow. |
| `.github` | The CI and release workflow, and the dependency update settings. |

A new edit needs three things before it is added: the code in the
decompilation that shows the game accepts it, a fixed range in
`internal/save`, and a check in the real game.

### Changelog and releases

Versions follow [Semantic Versioning](https://semver.org/) and are recorded in
[CHANGELOG.md](CHANGELOG.md), which follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Describe every
notable change under `## [Unreleased]` as part of the change itself.

The workflow in `.github/workflows/release.yml` checks every push to `main`
and every pull request. A push to `main` is also a release when the changelog
has a version that is not tagged yet. To release:

1. In `CHANGELOG.md`, add a `## [X.Y.Z] - YYYY-MM-DD` heading below
   `## [Unreleased]`, so that the unreleased entries now belong to the new
   version, and update the links at the bottom of the file.
2. Run `just release-status` to see the version and the notes that would be
   released.
3. Commit and push to `main`.

The workflow then builds the archives for every platform, runs each program on
its own platform, publishes the GitHub release with the notes of that version
and creates the `vX.Y.Z` tag on the pushed commit. A push that adds no new
version is only checked.

## Credits

The save format knowledge comes from
[zcanann/FFCC-Decomp](https://github.com/zcanann/FFCC-Decomp). The GCI container
layout and the folder-loading rules come from
[Dolphin](https://github.com/dolphin-emu/dolphin). The menu is drawn with
[huh](https://github.com/charmbracelet/huh) by Charm.

Maintained by [ailequal](https://github.com/ailequal).

## License

[MIT](LICENSE).

This is an unofficial fan project, not affiliated with or endorsed by Square
Enix or Nintendo. Final Fantasy Crystal Chronicles and the item names used
here belong to their respective owners.
