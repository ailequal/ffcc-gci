# ffcc-gci

A command-line save editor for **Final Fantasy Crystal Chronicles** (GameCube),
working on save files in Dolphin's `.gci` format.

> **PAL only.** Only the European version of the game (`GCCP01`) is supported.
> Saves from the USA and Japanese versions are refused.

## Goal

Edit only what is safe. An edit is added only when both of these hold:

- The [FFCC decompilation](https://github.com/zcanann/FFCC-Decomp) shows how the
  game reads the value and that it accepts the edited one.
- The result is a save the game could have produced by itself through normal
  play.

Anything that cannot be confirmed that way stays out.

Every value the tool can set has a fixed range, taken from the game's own
code. A value outside its range is refused and nothing is written.
`ffcc-gci limits` lists every range and the evidence behind it.

## Commands

| Command | What it does | Status |
|---|---|---|
| `info` | Show slots, characters, gil, stats, items and artifacts. Read-only. | Available |
| `set-gil` | Set a character's gil, from 0 to 99,999,999. | Available, tested in-game |
| `add-artifact` | Grant a character one of the game's 73 artifacts. | Available, tested in-game |
| `artifacts` | List the artifacts that can be granted, by what they improve. Read-only. | Available |
| `limits` | Show every editable value and the range it accepts. Read-only. | Available |
| `add-item` | Add Phoenix Down, a material, a food or a design to a character's inventory. | Available, in-game test pending |
| `items` | List the items that can be added, by group. Read-only. | Available |

Strength, magic, defence and hearts are not directly editable: the game
recomputes them from the character's tribe, equipment and artifacts
(`CCaravanWork::CalcStatus`). Artifacts are the way to raise them, as in normal
play.

## Usage

Close Dolphin first. It keeps the save in memory while a game runs and writes
it back, which would undo or clash with an edit.

```sh
# Look at a save
ffcc-gci info save.gci

# Give the character Alba in slot 3 some gil
ffcc-gci set-gil -slot 3 -char Alba -gil 50000 save.gci

# Same, but write a new file and leave the save untouched
ffcc-gci set-gil -slot 3 -char Alba -gil 50000 -o edited.gci save.gci

# See which artifacts exist, then grant one
ffcc-gci artifacts
ffcc-gci add-artifact -slot 3 -char Alba -artifact "Sun Pendant" save.gci

# See which items can be added, then add two of one
ffcc-gci items
ffcc-gci add-item -slot 3 -char Alba -item Mythril -count 2 save.gci
```

`-slot` and `-char` use the numbers `info` prints. `-char` also takes the
character's name.

Dolphin's GCI folder for PAL saves is usually `GC/EUR/Card A/` inside its user
directory. Saves on a raw memory card image (`.raw`) are not supported; export
them to `.gci` with Dolphin's memory card manager.

### Artifacts and stats

An artifact is either owned or not. Granting one the character already has
changes nothing, and nothing is written.

After an artifact is granted, the game counts it as soon as the save is
loaded: it works out strength, magic, defence, hearts and command slots from
the artifacts by itself. `ffcc-gci info` lists the new artifact straight away,
but its hearts and stat numbers are the ones the game last stored in the save,
so they stay the same until the game has saved again.

### Items

Only four groups of ordinary items can be added: Phoenix Down, materials, food
and designs, 164 items in all. `ffcc-gci items` lists them.

- Items do not stack. Each one takes one of the character's 64 inventory
  slots, and the command refuses to add more than there are free slots.
- Holding several of the same material, food or Phoenix Down is normal; the
  blacksmith even counts the copies of a material.
- A design is added only once, and only if it is for anyone or for the
  character's own tribe.

## Deliberately not supported

Safety comes first, so the tool refuses whatever it cannot prove safe. None of
the following can be done with it.

| Not supported | Why |
|---|---|
| Equipment (weapons, armour, accessories) | It is tied to a tribe, and the game does not re-check what a character has equipped. |
| Magicite and command items | They only exist inside a dungeon. The game deletes them when the party leaves (`CCaravanWork::SafeDeleteTempItem`), so no real save holds them. |
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

## Safety

- **Validation first.** The file must be a PAL save of this game with the
  expected size, and the slot must pass the checks the game applies on load
  (header tags, region, checksum). Otherwise nothing is edited.
- **The original is kept.** An in-place edit first copies the original to
  `<save.gci>.<date-time>.bak`. The backup does not end in `.gci` on purpose:
  Dolphin loads every `.gci` in a card folder and rejects duplicates. For the
  same reason, do not put a `-o` output next to the original in that folder.
- **Limits come from the game.** Each value is restricted to the range the
  game itself keeps it in, and the range is checked twice: when the command
  line is read and again when the value is stored.
- **Self-check before writing.** The edited file is decoded again and compared
  with the original; if anything other than the intended field and the
  checksum changed, nothing is written.
- **No half-written saves.** The new file is written beside the save and
  renamed over it in one step.

## Limits and why each edit is safe

| Value | Accepted range | Evidence in the decomp |
|---|---|---|
| Artifact | One of the 73 artifacts, item IDs 159 to 231 | Each character has one flag per artifact (`Mc::CharaDat::m_artifactFlags`), and setting a flag is all the game does when a character receives one (`CCaravanWork::SetArtifact`). On load the game rebuilds the artifact list from the flags (`CMemoryCardMan::SetLoadData`) and recomputes hearts; the other stats are recomputed on every frame a character is on screen (`CGPartyObj::onFrameAlways`). The game caps the totals: stats at 99, hearts at 8, command slots at 8 (`CCaravanWork::CalcStatus`). The 23 extra flags are unnamed placeholders and are refused. |
| Item | Phoenix Down (293), materials (294 to 351), food (381 to 392), designs (401 to 493) | Every one of these IDs is described by the game itself as "Revives the fallen", "Material for equipment", "Restores HP" or a design (`gItemDescIds` in the GBA client), and the designs are all working recipes in its item table. Adding one does what the game does in `CCaravanWork::AddItem`: fill the first free slot and raise the item count, which is how road events hand out items on the world map. |
| Items held | 0 to 64 per character | The game refuses to add an item when all 64 slots are used (`CCaravanWork::AddItem`). On load the PAL version recounts the inventory (`CMemoryCardMan::SetLoadData`). |
| Gil | 0 to 99,999,999 | Stored as a 32-bit integer per character (`Mc::CharaDat::m_gil`), copied as-is on load (`CMemoryCardMan::SetLoadData`). The game keeps it within 0 and 99,999,999 (`CCaravanWork::AddGil`, `CanAddGil`). |

## Build and test

Requires Go.

```sh
make build     # writes bin/ffcc-gci
make check     # vet, tests and formatting
```

The tests build synthetic saves in memory. Real saves are never committed:
they are personal data and embed the game's banner art. To also run the tests
against a real PAL save, put one at `testdata/pal.gci` (git-ignored).

## Layout

| Path | Responsibility |
|---|---|
| `cmd/ffcc-gci` | Program entry point. |
| `internal/cli` | Argument parsing, command dispatch, output, file writing. |
| `internal/gci` | The GCI container (header and blocks). |
| `internal/save` | Save slots: decoding, checksum, character fields, edit verification. |
| `internal/save/savetest` | Synthetic saves for tests. |
| `internal/itemdb` | Item IDs, names and the game's own description of each item. |

## Save format

The file is 22 memory card blocks: a `0x4000`-byte banner area followed by four
save slots of `0xA000` bytes. In each slot, everything after the first `0x18`
bytes is stored as little-endian 32-bit words rotated left by a per-slot key,
and the decoded data carries a CRC-32 (polynomial `0x04C11DB7`).

## Credits

The save format knowledge comes from
[zcanann/FFCC-Decomp](https://github.com/zcanann/FFCC-Decomp). The GCI container
layout and the folder-loading rules come from
[Dolphin](https://github.com/dolphin-emu/dolphin).

## License

[MIT](LICENSE).

This is an unofficial fan project, not affiliated with or endorsed by Square
Enix or Nintendo. Final Fantasy Crystal Chronicles and the item names used
here belong to their respective owners.
