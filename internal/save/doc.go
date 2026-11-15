// Package save models the game's save data inside a GCI payload: the
// three save slots, their obfuscation, their checksum and the fields of
// each character record.
//
// Layout and algorithms follow the FFCC decompilation project
// (https://github.com/zcanann/FFCC-Decomp), mainly src/memorycard.cpp and
// include/ffcc/memorycard.h. Every editable field must cite the decomp
// code that shows the game accepts the edited value.
package save
