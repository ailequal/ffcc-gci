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

## What it can do

| Edit | Command | Accepted values |
|---|---|---|
| Show a save: slots, characters, gil, stats, items, artifacts | `info` | Read-only |
| Set a character's gil | `set-gil` | 0 to 99,999,999 |
| Grant a character an artifact | `add-artifact` | The game's 73 artifacts |
| Add items to a character's inventory | `add-item` | Phoenix Down, materials, food and designs: 164 items, up to 64 per character |

Every edit was confirmed in the real game on a PAL save.

Strength, magic, defence and hearts cannot be set directly, because the game
recalculates them. Artifacts are the way to raise them, as in normal play.

## Install

**Download a release.** The
[releases page](https://github.com/ailequal/ffcc-gci/releases) has a ready-made
program for Linux, macOS and Windows. Unpack the archive for your system and
run `ffcc-gci help`. Each release lists how to verify the download.

**Or install it with Go.** You need [Go](https://go.dev/dl/) 1.27.3 or newer.
This makes `ffcc-gci` run from any folder:

```sh
go install github.com/ailequal/ffcc-gci/cmd/ffcc-gci@latest
```

**Or build it from a clone of this repository:**

```sh
make build      # writes bin/ffcc-gci
make install    # or install it into Go's bin directory
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
  `GC/EUR/Card A/` inside Dolphin's user directory. Saves on a raw memory card
  image (`.raw`) are not supported; export them to `.gci` with Dolphin's
  memory card manager.
- **Know how to undo.** An edit first copies the original to
  `<save.gci>.<date-time>.bak` next to the save. To undo, copy that file back
  over the save. With `-o <new file>` the save is not touched at all.
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
| [docs/USAGE.md](docs/USAGE.md) | Every command, flag, example and accepted value, with the full lists of artifacts and items. |
| [docs/SAVE_FORMAT.md](docs/SAVE_FORMAT.md) | The save file format, what the game recalculates, and the evidence that makes each edit safe. |
| [CHANGELOG.md](CHANGELOG.md) | What changed in each version. |

## Getting help

Start with `ffcc-gci help <command>` and [docs/USAGE.md](docs/USAGE.md). For a
problem or a question, open an issue at
<https://github.com/ailequal/ffcc-gci/issues>. If the game rejects an edited
save, restore the `.bak` file and include the output of `ffcc-gci info` on it.

## Development

```sh
make check        # vet, tests, formatting and the changelog format; run it before every commit
make lint         # golangci-lint, if you have it installed
make lint-docker  # the same with the pinned release in Docker, as the workflow runs it
make docs         # regenerate docs/USAGE.md after changing a command
make dist         # cross-compile the release archives into dist/
```

Each command is described once, in `internal/cli/commands.go`. The help the
program prints and `docs/USAGE.md` are both rendered from that description,
and a test fails when the document is out of date. Another test runs every
example in it against a sample save.

The tests build synthetic saves in memory. Real saves are never committed:
they are personal data and embed the game's banner art. To also run the tests
against a real PAL save, put one at `testdata/pal.gci` (git-ignored).

| Path | Responsibility |
|---|---|
| `cmd/ffcc-gci` | Program entry point. |
| `internal/cli` | Commands, help, output and file writing. |
| `internal/gci` | The GCI container (header and blocks). |
| `internal/save` | Save slots: decoding, checksum, character fields, limits, edit verification. |
| `internal/save/savetest` | Synthetic saves for tests. |
| `internal/itemdb` | Item IDs, names and the game's own description of each item. |
| `docs` | The usage reference and the save format notes. |
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
2. Commit and push to `main`.

The workflow then builds the archives for every platform, runs each program on
its own platform, publishes the GitHub release with the notes of that version
and creates the `vX.Y.Z` tag on the pushed commit. A push that adds no new
version is only checked.

## Credits

The save format knowledge comes from
[zcanann/FFCC-Decomp](https://github.com/zcanann/FFCC-Decomp). The GCI container
layout and the folder-loading rules come from
[Dolphin](https://github.com/dolphin-emu/dolphin).

Maintained by [ailequal](https://github.com/ailequal).

## License

[MIT](LICENSE).

This is an unofficial fan project, not affiliated with or endorsed by Square
Enix or Nintendo. Final Fantasy Crystal Chronicles and the item names used
here belong to their respective owners.
