# ffcc-gci

A command-line save editor for **Final Fantasy Crystal Chronicles** (GameCube),
working on save files in Dolphin's `.gci` format.

**Status:** project skeleton only. No save-editing command exists yet.

## Goal

Edit only what is safe. A feature is added only when the
[FFCC decompilation](https://github.com/zcanann/FFCC-Decomp) shows how the game
reads the value and that it accepts the edited one. Anything that cannot be
confirmed that way stays out.

## Planned commands

| Command | What it does |
|---|---|
| `info` | Show slots, characters, gil, stats, items and artifacts. Read-only. |
| `items` | Search the item table by name or ID. Read-only. |
| `set-gil` | Set a character's gil. |
| `add-item` | Add an allowlisted item to a free inventory slot. |
| `add-artifact` | Grant a character an artifact. |

Strength, magic, defense and max HP are not directly editable: the game
recomputes them from the character's tribe, equipment and artifacts.

## Safety rules

- The input file is validated (game code, size, slot checksum) before any edit.
- An edit never destroys the original: a backup is always kept.
- Values are limited to what the game itself allows (for example the gil cap).

## Build

Requires Go.

```sh
make build     # writes bin/ffcc-gci
make check     # vet, tests and formatting
```

## Layout

| Path | Responsibility |
|---|---|
| `cmd/ffcc-gci` | Program entry point. |
| `internal/cli` | Argument parsing, command dispatch, output. |
| `internal/gci` | The GCI container (header and blocks). |
| `internal/save` | Save slots: decoding, checksum, character fields. |
| `internal/itemdb` | Item IDs, names, categories and the allowlist. |

## Credits

The save format knowledge comes from
[zcanann/FFCC-Decomp](https://github.com/zcanann/FFCC-Decomp).
