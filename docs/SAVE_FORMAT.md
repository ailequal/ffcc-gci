# Save format and safety evidence

This page records what ffcc-gci knows about the save file of Final Fantasy
Crystal Chronicles (GameCube, PAL), and why each edit it makes is safe.

Everything here comes from the
[FFCC decompilation](https://github.com/zcanann/FFCC-Decomp) and from
[Dolphin](https://github.com/dolphin-emu/dolphin). Function and file names in
`code style` refer to those projects. All multi-byte values are big-endian
unless stated otherwise.

## The GCI container

A `.gci` file is the save's 64-byte memory card directory entry followed by its
data blocks of `0x2000` bytes each (`DEntry` in Dolphin's `GCMemcard.h`).

| Offset | Size | Field | Value for this game |
|---|---|---|---|
| `0x00` | 4 | Game code | `GCCP` |
| `0x04` | 2 | Maker code | `01` |
| `0x08` | 32 | File name, NUL padded | `FFCC` |
| `0x38` | 2 | Number of blocks | 22 |

The container has no checksum. Dolphin accepts a GCI only if its size is
exactly the header plus the declared blocks, and ffcc-gci applies the same
rule. Dolphin loads every `.gci` file in a card folder and rejects a second
one with the same game and file name (`GCMemcardDirectory::LoadGCI`), which is
why backups are named `.bak`.

## The save data

The 22 blocks (`0x2C000` bytes) hold a banner area and four save slots.

| Offset | Size | Content |
|---|---|---|
| `0x0000` | `0x4000` | Comment, banner and icons |
| `0x4000 + n × 0xA000` | `0xA000` | Save slot *n*, for *n* = 0 to 3 |

The game reads and writes slot *n* at that offset (`McCtrl` in `wm_menu.cpp`).
A slot that was never used is all zero.

### Slot header

| Offset | Size | Field | Notes |
|---|---|---|---|
| `0x00` | 4 | Maker tag | `GDS` |
| `0x04` | 4 | Title tag | `FFCC` |
| `0x08` | 4 | Machine tag | `GC` |
| `0x0C` | 4 | Version tag | `1.00` |
| `0x10` | 1 | Region | `E` on PAL |
| `0x11` | 1 | Rotation key | Random, chosen at each save |
| `0x18` | 4 | Random value | Chosen at each save |
| `0x1C` | 4 | Checksum | See below |

### Obfuscation

The first `0x18` bytes of a slot are stored as they are. Everything after is
handled as 32-bit words: the game rotates each word left by the rotation key
modulo 32 and stores it little-endian (`CMemoryCardMan::EncodeData`). Decoding
reads each word little-endian and rotates it right by the same amount
(`DecodeData`).

ffcc-gci keeps the key and the random value of a slot as they are.

### Checksum

The checksum is a CRC-32 with polynomial `0x04C11DB7`, processed most
significant bit first, starting from `0xFFFFFFFF` and complemented at the end
(`CMemoryCardMan::CalcCrc`). It is computed on the decoded slot, over bytes
`0x00` to `0x1B` and `0x20` to `0x8BCF`. The rest of the slot is padding.

### What the game checks before loading a slot

The game lists and loads a slot only if all of these hold
(`McCtrl::SetListDat`, `CMemoryCardMan::SetLoadData`):

- the four tags and the region have the values above;
- the checksum matches;
- the town name is not empty.

ffcc-gci applies the same checks before it edits a slot, and again to its own
output before it writes.

### Slot fields

Offsets are in the decoded slot (`Mc::SaveDat` in `memorycard.h`).

| Offset | Size | Field | Notes |
|---|---|---|---|
| `0x2C` | 4 | Play time in frames | 1500 frames are one minute on PAL |
| `0x30` | 16 | Party | Four character indices, -1 for none |
| `0x10C0` | 16 | Town name | NUL padded |
| `0x14D0` | 8 × `0x9C0` | Character records | |

### Character record

Offsets are in a character record (`Mc::CharaDat`).

| Offset | Size | Field | Notes |
|---|---|---|---|
| `0x06` | 2 | Maximum hit points | In half hearts |
| `0x08` | 2 | Hit points | In half hearts |
| `0x0A` | 2 | Strength | |
| `0x0C` | 2 | Magic | |
| `0x0E` | 2 | Defence | |
| `0x28` | 2 | Number of items held | |
| `0x2E` | 2 | Tribe | 0 Clavat, 1 Lilty, 2 Yuke, 3 Selkie |
| `0x34` | 4 × 2 | Equipment | Inventory slot of each equipped item, -1 for none |
| `0x3C` | 64 × 2 | Inventory | Item ID per slot, -1 for an unused slot |
| `0xBC` | 3 × 4 | Artifact flags | Flag *n* is bit *n* mod 32 of word *n* / 32 |
| `0xEC` | 4 | Gil | Signed |
| `0xF0` | 16 | Name | NUL padded |
| `0x5B4` | 4 | In use | Non-zero when the record holds a character |

ffcc-gci writes only the inventory, the number of items, the artifact flags
and the gil.

## What the game recalculates

The strength, magic, defence and hit points in a record are the game's last
calculation, not values it trusts.

- **Maximum hit points** are recomputed from the artifacts when a save is
  loaded (`CMemoryCardMan::SetLoadData` calling
  `CCaravanWork::GetArtifactIncludeHpMax`).
- **Strength, magic, defence and command slots** are recomputed from tribe,
  equipment and artifacts on every frame a character is on screen
  (`CGPartyObj::onFrameAlways` calling `CCaravanWork::CalcStatus`).
- Both the menu on the television and the GBA screens show stats only for a
  character that is on screen, so the game never shows the stored numbers.

This is why ffcc-gci does not edit those numbers, and why `ffcc-gci info`
keeps showing the old ones after an artifact is granted, until the game has
saved again.

## Item IDs

The game's item menu sorts item IDs into fixed ranges
(`CMenuPcs::GetItemType`), and its GBA client holds a description for every
item (`gItemDescIds` and `gItemDescs_En` in `gba/src/cli/main/msg.c`).

| Item IDs | What they are | Can be added |
|---|---|---|
| 1 to 158 | Equipment | No |
| 159 to 231 | Artifacts | Granted as flags, never as inventory items |
| 232 to 255 | Unused | No |
| 256 to 292 | Magicite and command items | No |
| 293 | Phoenix Down | Yes |
| 294 to 351 | Materials | Yes |
| 352, 353 | Goddess Stat., Devil's Mask | No |
| 354 | Unused | No |
| 355 to 363 | Seeds | No |
| 364 to 367 | Story items | No |
| 368 | Unused | No |
| 369, 370 | Ultimite, Dark Sphere | No |
| 371 to 380 | Unused | No |
| 381 to 392 | Food | Yes |
| 393 to 400 | Shards, Wheat, Flour and others | No |
| 401 to 493 | Designs | Yes |
| 494 to 500 | Unused | No |

## Why each edit is safe

An edit is supported only when the decompilation shows how the game reads the
value, and the result is a save the game could have produced by itself.

| Edit | What is written | Evidence |
|---|---|---|
| Gil | The 32-bit gil field, 0 to 99,999,999 | The field is copied as it is on load (`SetLoadData`). The game keeps gil in this range: `CCaravanWork::AddGil` clamps to it and `CanAddGil` refuses to go past it. |
| Artifact | One artifact flag | Setting the flag is all the game does when a character receives an artifact (`CCaravanWork::SetArtifact`); on load it rebuilds the artifact list from the flags. A flag is on or off, so an artifact cannot be held twice. The game caps the totals in `CalcStatus`: stats at 99, hearts at 8, command slots at 8. |
| Item | One inventory slot and the number of items | `CCaravanWork::AddItem` fills the first unused slot and raises the count by one, and refuses at 64 items. Road events on the world map hand out items through it, and the world map is where the game saves. On load the PAL version recounts the inventory. |

Which items can be added is decided per item ID, from the game's own
description of it: "Revives the fallen", "Material for equipment", "Restores
HP" or a design. A test fails if an addable range ever contains anything else.

- **Copies of a material, a food or Phoenix Down** are normal. The blacksmith
  counts the copies of a material a character holds
  (`SmithForgeScreen_PrintNextRow` in the GBA client), and saves made by the
  game hold several of one food.
- **Designs** are all working recipes: rows 401 to 493 of the item table. The
  forge screen refuses to forge a design whose result the character cannot
  equip (`Item_CanEquip`, in the same function), so ffcc-gci adds only designs
  for anyone or for the character's own tribe, and only one copy.

Before any edit is written, the edited file is decoded again and compared with
the original. If anything other than the intended bytes and the checksum
differs, nothing is written (`save.VerifyEdit`).

## Tested in the game

Each edit was confirmed on a real PAL save in Dolphin:

- gil set to 50,000 showed in the game;
- a granted Sun Pendant appeared in the artifact list and added a heart;
- added Mythril appeared in the item list, and the blacksmith offered Iron
  Plate from an added Iron Armour design.
