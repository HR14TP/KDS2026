package main

import (
	"testing"

	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The D03 fix: a Sally (POS) update naming an item/subitem by name alone is
// ambiguous whenever an order has more than one item sharing that name (two
// identical steaks, say) - getItemSally/getSubItemSally now disambiguate by
// occurrence (1-based rank by id ASC among same-named siblings), the same
// pattern getDatabaseItemByOccurrence already uses for every other
// cross-terminal item lookup in this codebase. Reuses
// setupCloseOrderTestDB/mustCreateOrder from closeorder_scope_test.go - same
// package, same isolated in-memory DB pattern.

func TestGetItemSally_DisambiguatesByOccurrenceNotJustName(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	order := mustCreateOrder(t, db, "ref-two-steaks")
	screen := mustCreateScreen(t, db, "kitchen", "")
	first := mustCreateItem(t, db, order.ID, screen.ID, "rump steak 300gr", "OPEN")
	second := mustCreateItem(t, db, order.ID, screen.ID, "rump steak 300gr", "OPEN")

	got1 := getItemSally(order.ID, kdsmodels.SallyMainItems{Name: "rump steak 300gr"}, 1)
	assert.Equal(t, first.ID, got1.ID, "occurrence 1 must resolve to the earliest-created sibling")

	got2 := getItemSally(order.ID, kdsmodels.SallyMainItems{Name: "rump steak 300gr"}, 2)
	assert.Equal(t, second.ID, got2.ID, "occurrence 2 must resolve to the second sibling, not the first")

	got3 := getItemSally(order.ID, kdsmodels.SallyMainItems{Name: "rump steak 300gr"}, 3)
	assert.Equal(t, 0, got3.ID, "occurrence with no matching sibling must be a clean miss (ID 0), not a wrong match")
}

func TestGetItemSally_QtyChangeInPlaceStillMatchesSameItem(t *testing.T) {
	// Deliberately NOT scoped by Qty (unlike getDatabaseItemByOccurrence) -
	// a qty change on an existing single instance must still resolve to
	// that same item, not be treated as a miss.
	db := setupCloseOrderTestDB(t)
	order := mustCreateOrder(t, db, "ref-qty-bump")
	screen := mustCreateScreen(t, db, "kitchen", "")
	item := mustCreateItem(t, db, order.ID, screen.ID, "burger", "OPEN")

	got := getItemSally(order.ID, kdsmodels.SallyMainItems{Name: "burger", Qty: 2}, 1)
	assert.Equal(t, item.ID, got.ID, "a qty change must still match the existing single instance by name+occurrence")
}

func TestGetSubItemSally_DisambiguatesByOccurrenceNotJustName(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	order := mustCreateOrder(t, db, "ref-sub-occurrence")
	screen := mustCreateScreen(t, db, "kitchen", "")
	item := mustCreateItem(t, db, order.ID, screen.ID, "burger", "OPEN")
	firstSub := kdsmodels.KDSSubitems{KDSItemID: item.ID, Name: "no onion", Status: "OPEN"}
	require.NoError(t, db.Create(&firstSub).Error)
	secondSub := kdsmodels.KDSSubitems{KDSItemID: item.ID, Name: "no onion", Status: "OPEN"}
	require.NoError(t, db.Create(&secondSub).Error)

	got1 := getSubItemSally(item.ID, kdsmodels.SallySubItems{Name: "no onion"}, 1)
	assert.Equal(t, firstSub.ID, got1.ID)

	got2 := getSubItemSally(item.ID, kdsmodels.SallySubItems{Name: "no onion"}, 2)
	assert.Equal(t, secondSub.ID, got2.ID)
}

// This is the exact D03 bug scenario: two identical main items already on
// an order (e.g. two "rump steak 300gr"), and a single incoming Sally round
// attaches a modifier to only the SECOND one. Before the fix, getItemSally
// matched by name alone with no ordering, so the modifier could land on
// either instance depending on SQLite's unordered return - confirmed broken
// by construction (name-only WHERE, no ORDER BY, First()). After the fix,
// occurrence (this item's 1-based rank among same-named items in THIS same
// incoming payload) must land on the correct instance every time.
func TestProcessSallyMainItem_ModifierAttachesToCorrectOccurrence(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	order := mustCreateOrder(t, db, "ref-modifier-occurrence")
	screen := mustCreateScreen(t, db, "kitchen", "")
	firstSteak := mustCreateItem(t, db, order.ID, screen.ID, "rump steak 300gr", "OPEN")
	secondSteak := mustCreateItem(t, db, order.ID, screen.ID, "rump steak 300gr", "OPEN")

	sallyOrder := kdsmodels.SallyOrder{
		ProductCenter: "kitchen",
		Items: []kdsmodels.SallyMainItems{
			{Name: "rump steak 300gr", Qty: 1, SubItems: []kdsmodels.SallySubItems{{Name: "medium", Qty: 1}}},
			{Name: "rump steak 300gr", Qty: 1, SubItems: []kdsmodels.SallySubItems{{Name: "well done", Qty: 1}}},
		},
	}
	processSallyMainItem(sallyOrder, order.ID, false)

	var firstSubs []kdsmodels.KDSSubitems
	require.NoError(t, db.Where("kds_item_id = ?", firstSteak.ID).Find(&firstSubs).Error)
	require.Len(t, firstSubs, 1, "the first steak must receive exactly one modifier")
	assert.Equal(t, "medium", firstSubs[0].Name)

	var secondSubs []kdsmodels.KDSSubitems
	require.NoError(t, db.Where("kds_item_id = ?", secondSteak.ID).Find(&secondSubs).Error)
	require.Len(t, secondSubs, 1, "the second steak must receive exactly one modifier")
	assert.Equal(t, "well done", secondSubs[0].Name, "the second steak's own modifier must not land on the first steak instead")
}

// The real-world shape of the bug: round 1 creates two plain steaks with no
// modifiers; round 2 (a separate, later Sally update - e.g. the waiter adds
// a modifier after the fact) attaches "well done" to only the second one.
// The first steak must come out of round 2 exactly as it went in.
func TestProcessSallyMainItem_SecondRoundModifierOnlyAffectsTargetedInstance(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	order := mustCreateOrder(t, db, "ref-second-round")
	screen := mustCreateScreen(t, db, "kitchen", "")
	firstSteak := mustCreateItem(t, db, order.ID, screen.ID, "rump steak 300gr", "OPEN")
	secondSteak := mustCreateItem(t, db, order.ID, screen.ID, "rump steak 300gr", "OPEN")

	sallyOrder := kdsmodels.SallyOrder{
		ProductCenter: "kitchen",
		Items: []kdsmodels.SallyMainItems{
			{Name: "rump steak 300gr", Qty: 1},
			{Name: "rump steak 300gr", Qty: 1, SubItems: []kdsmodels.SallySubItems{{Name: "well done", Qty: 1}}},
		},
	}
	processSallyMainItem(sallyOrder, order.ID, false)

	var firstSubs []kdsmodels.KDSSubitems
	require.NoError(t, db.Where("kds_item_id = ?", firstSteak.ID).Find(&firstSubs).Error)
	assert.Len(t, firstSubs, 0, "the untouched first steak must not have picked up the second steak's modifier")

	var secondSubs []kdsmodels.KDSSubitems
	require.NoError(t, db.Where("kds_item_id = ?", secondSteak.ID).Find(&secondSubs).Error)
	require.Len(t, secondSubs, 1)
	assert.Equal(t, "well done", secondSubs[0].Name)
}

// A genuinely new second instance of an existing name (e.g. the table orders
// a second round of fries) must create a new item, not silently merge into
// the first - occurrence 2 has no sibling yet, so getItemSally must miss
// and fall through to item creation.
func TestProcessSallyMainItem_NewInstanceOfExistingNameCreatesSeparateItem(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	order := mustCreateOrder(t, db, "ref-reorder")
	screen := mustCreateScreen(t, db, "kitchen", "")
	mustCreateItem(t, db, order.ID, screen.ID, "fries", "OPEN")

	sallyOrder := kdsmodels.SallyOrder{
		ProductCenter: "kitchen",
		Items: []kdsmodels.SallyMainItems{
			{Name: "fries", Qty: 1},
			{Name: "fries", Qty: 1},
		},
	}
	processSallyMainItem(sallyOrder, order.ID, false)

	var items []kdsmodels.KDSItems
	require.NoError(t, db.Where("kds_order_id = ? AND name = ?", order.ID, "fries").Find(&items).Error)
	assert.Len(t, items, 2, "a genuinely new second instance must be created, not merged into the existing one")
}
