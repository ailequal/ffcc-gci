# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

<!--
To release: add a "## [X.Y.Z] - YYYY-MM-DD" heading below "## [Unreleased]",
so that the entries become part of the new version, and update the links at
the bottom: point [unreleased] to .../compare/vX.Y.Z...HEAD and add
[X.Y.Z]: .../releases/tag/vX.Y.Z (or .../compare/vPREVIOUS...vX.Y.Z).
Pushing that commit to main publishes the release and creates the tag.
-->

## [Unreleased]

### Changed

- Bubble Tea, the library that draws the menu, updated from 2.1.1 to 2.1.3.
- Go 1.27.4 required to build from source, up from 1.27.3; the release archives are built with it.

### Fixed

- Menu tests that failed on macOS.

### Security

- `menu` no longer ends with a panic when the folder it browses holds a file whose name has a malformed escape sequence.

## [1.0.0] - 2026-11-16

The first stable version. It edits saves of the European (PAL) version of the
game, and only in the ways the game's own code shows to be safe.

### Added

- `info` command that shows the four save slots of a PAL save and every character with gil, hearts, stats, items and artifacts, without changing the file.
- `set-gil` command that sets a character's gil, from 0 to 99,999,999.
- `add-artifact` command that grants a character one of the game's 73 artifacts, and `artifacts` command that lists them by what they improve.
- `add-item` command that adds Phoenix Down, materials, food and designs to a character's inventory, 164 items in all, and `items` command that lists them.
- `menu` command that makes the same edits by asking questions instead of taking flags: it finds the save in Dolphin's folders, lists only what a character can receive, and runs the `set-gil`, `add-artifact` or `add-item` command after a confirmation. It is built into the program and needs nothing else installed.
- `limits` command that shows every value the program can set, its accepted range and the evidence that the range is safe.
- `help` command with the flags and examples of each command, `help -all` for every command at once and `help -markdown` for the reference in Markdown.
- `version` command that prints the version, the Go toolchain and the platform.
- Validation of a save before any edit: game code, file size, and the header tags, region and checksum of the slot, as the game checks them on load.
- Backup of the original save as `<save.gci>.<date-time>.bak` before every edit in place, numbered when several fall in the same second, and `-o` to write a new file instead.
- Check of every edit before it is written: the edited file is decoded again, and nothing is written if anything but the intended bytes and the checksum changed.
- Step-by-step guide in `docs/GUIDE.md`, usage reference in `docs/USAGE.md`, generated from the program's own help, and save format notes in `docs/SAVE_FORMAT.md`.
- Release archives for Linux, macOS and Windows on amd64 and arm64, built by a workflow when a new version in this file reaches the `main` branch, smoke-tested on each platform and published with SHA-256 checksums and build provenance attestations.

[unreleased]: https://github.com/ailequal/ffcc-gci/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/ailequal/ffcc-gci/releases/tag/v1.0.0
