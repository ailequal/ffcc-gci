// Package gci reads and writes the GCI container: the 0x40-byte GameCube
// directory entry that Dolphin stores in front of a save file's raw
// blocks.
//
// The package only knows about the container. It validates the header
// (game code, block count, file size) and hands the payload to the save
// package; it does not interpret the game's own data.
package gci
