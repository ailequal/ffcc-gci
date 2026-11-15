package save

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

// Item IDs that can be added to an inventory. Each range is a group of
// ordinary items, identified one by one through the game's own item
// descriptions (see the itemdb package). IDs between and around the
// ranges are equipment, magicite, seeds, story items and placeholders,
// which are left out on purpose.
const (
	PhoenixDown   = 293
	FirstMaterial = 294 // Bronze
	LastMaterial  = 351 // Remedy
	FirstFood     = 381 // Striped Apple
	LastFood      = 392 // Strange Liquid
	FirstDesign   = 401 // Novice's Wpn.
	LastDesign    = 493 // Ring of Invin.
)

var (
	// ErrInventoryFull reports that a character has no free inventory
	// slot left.
	ErrInventoryFull = errors.New("the inventory is full")
	// ErrInventoryCount reports a save whose item count does not match
	// its inventory.
	ErrInventoryCount = errors.New("the inventory does not match its item count")
)

// ItemLimits are the ranges of item IDs that can be added to an
// inventory, one per group.
var ItemLimits = []Limit{
	{
		Name: "Phoenix Down item ID", Min: PhoenixDown, Max: PhoenixDown,
		Source: "The game describes this item as \"Revives the fallen\". Saves made by the game " +
			"hold several at once.",
	},
	{
		Name: "material item ID", Min: FirstMaterial, Max: LastMaterial,
		Source: "The game describes each of these 58 items as \"Material for equipment\". The " +
			"blacksmith counts how many of a material a character holds, so holding several of " +
			"one is how the game is meant to work (SmithForgeScreen_PrintNextRow in the GBA client).",
	},
	{
		Name: "food item ID", Min: FirstFood, Max: LastFood,
		Source: "The game describes each of these 12 items as \"Restores HP\" and its item menu " +
			"puts them in one category (CMenuPcs::GetItemType). Saves made by the game hold " +
			"several of one food.",
	},
	{
		Name: "design item ID", Min: FirstDesign, Max: LastDesign,
		Source: "The game describes each of these 93 items as a design, and all of them are " +
			"working recipes in its item table. A character can be given a design only once, and " +
			"only one for anyone or for the character's own tribe.",
	},
}

// InventoryLimit is the range of the number of items a character holds.
var InventoryLimit = Limit{
	Name: "items held by a character",
	Min:  0,
	Max:  InventorySize,
	Source: "A character has 64 inventory slots and the game refuses to add an item when all " +
		"are used (CCaravanWork::AddItem). Items do not stack: each one takes a slot.",
}

// CheckItem returns an error wrapping ErrOutOfRange when the item with
// the given ID is not one that can be added to an inventory.
func CheckItem(id int) error {
	ranges := make([]string, len(ItemLimits))
	for i, l := range ItemLimits {
		if l.Check(id) == nil {
			return nil
		}
		ranges[i] = fmt.Sprintf("%d-%d", l.Min, l.Max)
		if l.Min == l.Max {
			ranges[i] = fmt.Sprint(l.Min)
		}
	}
	return fmt.Errorf("%w: item ID %d cannot be added; the allowed IDs are %s",
		ErrOutOfRange, id, strings.Join(ranges, ", "))
}

// IsDesign reports whether the item ID is a design that can be added.
func IsDesign(id int) bool {
	return id >= FirstDesign && id <= LastDesign
}

// usedSlots counts the inventory slots that hold an item, the way the
// game counts them when it loads a save (SetLoadData).
func (c *Character) usedSlots() int {
	used := 0
	for _, id := range c.Inventory() {
		if id != EmptyItem {
			used++
		}
	}
	return used
}

// FreeSlots returns the number of unused inventory slots.
func (c *Character) FreeSlots() int {
	return InventorySize - c.usedSlots()
}

// HasItem reports whether the inventory holds the item with the given ID.
func (c *Character) HasItem(id int) bool {
	for _, held := range c.Inventory() {
		if held == id {
			return true
		}
	}
	return false
}

// ItemRange returns where an inventory slot is stored in the decoded
// slot.
func (c *Character) ItemRange(slot int) Range {
	start := offCharacters + c.index*characterSize + offInventory + 2*slot
	return Range{Start: start, End: start + 2}
}

// ItemCountRange returns where the character's item count is stored in
// the decoded slot.
func (c *Character) ItemCountRange() Range {
	start := offCharacters + c.index*characterSize + offItemCount
	return Range{Start: start, End: start + 2}
}

// AddItem adds one item to the character's inventory and returns the
// inventory slot it went to. It does what the game does in
// CCaravanWork::AddItem: put the item in the first unused slot and raise
// the item count by one. The game uses that routine for the items road
// events hand out on the world map, where it also saves.
//
// The item must pass CheckItem. A design the character already holds is
// refused with ErrAlreadyOwned, and a full inventory with
// ErrInventoryFull.
func (c *Character) AddItem(id int) (slot int, err error) {
	if !c.Exists() {
		return 0, fmt.Errorf("character record %d is not in use", c.index)
	}
	if err := CheckItem(id); err != nil {
		return 0, err
	}

	used := c.usedSlots()
	if used != c.ItemCount() {
		return 0, fmt.Errorf("%w: %d slots are used but the count says %d", ErrInventoryCount, used, c.ItemCount())
	}
	if IsDesign(id) && c.HasItem(id) {
		return 0, ErrAlreadyOwned
	}
	if used >= InventorySize {
		return 0, ErrInventoryFull
	}
	if err := InventoryLimit.Check(used + 1); err != nil {
		return 0, err
	}

	for slot, held := range c.Inventory() {
		if held == EmptyItem {
			binary.BigEndian.PutUint16(c.b[offInventory+2*slot:], uint16(id))
			binary.BigEndian.PutUint16(c.b[offItemCount:], uint16(used+1))
			return slot, nil
		}
	}
	return 0, ErrInventoryFull
}
