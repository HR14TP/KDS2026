package main

import (
	"testing"

	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The D06 fix: items added to an order that already exists on a peer
// terminal's own copy must actually reach that peer. Reuses
// setupCloseOrderTestDB/mustCreateScreen/mustCreateOrder from
// closeorder_scope_test.go - same package, same isolated in-memory DB
// pattern.

func TestApplyNewOrderItemsRemote_AppendsToExistingOrder(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	mustCreateScreen(t, db, "bar", "")
	order := mustCreateOrder(t, db, "K07TEST-001")

	payload := kdsmodels.KDSNewItemsOnOrder{
		Reference: "K07TEST-001",
		Items: []kdsmodels.KDSItems{
			{
				Name:   "sprite 300ml",
				Qty:    1,
				Status: "OPEN",
				Screen: kdsmodels.KDSScreens{Name: "bar"},
			},
		},
	}
	applyNewOrderItemsRemote(payload)

	var items []kdsmodels.KDSItems
	require.NoError(t, db.Where("kds_order_id = ?", order.ID).Find(&items).Error)
	require.Len(t, items, 1)
	assert.Equal(t, "sprite 300ml", items[0].Name)
	assert.Equal(t, "OPEN", items[0].Status)

	var screen kdsmodels.KDSScreens
	require.NoError(t, db.Where("id = ?", items[0].KDSScreenID).First(&screen).Error)
	assert.Equal(t, "bar", screen.Name, "item must be linked to the screen resolved by name, not a raw remote id")
}

func TestApplyNewOrderItemsRemote_CreatesScreenOnFirstSight(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	order := mustCreateOrder(t, db, "ref-no-screen-yet")

	payload := kdsmodels.KDSNewItemsOnOrder{
		Reference: "ref-no-screen-yet",
		Items: []kdsmodels.KDSItems{
			{Name: "margherita", Qty: 1, Status: "OPEN", Screen: kdsmodels.KDSScreens{Name: "pizza"}},
		},
	}
	applyNewOrderItemsRemote(payload)

	var screen kdsmodels.KDSScreens
	require.NoError(t, db.Where("LOWER(name) = LOWER(?)", "pizza").First(&screen).Error, "getScreenSally must create the screen locally on first sight")

	var items []kdsmodels.KDSItems
	require.NoError(t, db.Where("kds_order_id = ?", order.ID).Find(&items).Error)
	require.Len(t, items, 1)
	assert.Equal(t, screen.ID, items[0].KDSScreenID)
}

func TestApplyNewOrderItemsRemote_CreatesSubitems(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	mustCreateScreen(t, db, "kitchen", "")
	order := mustCreateOrder(t, db, "ref-with-subs")

	payload := kdsmodels.KDSNewItemsOnOrder{
		Reference: "ref-with-subs",
		Items: []kdsmodels.KDSItems{
			{
				Name:   "ribeye steak",
				Qty:    1,
				Status: "OPEN",
				Screen: kdsmodels.KDSScreens{Name: "kitchen"},
				SubItems: []kdsmodels.KDSSubitems{
					{Name: "medium rare", Qty: 1, Status: "OPEN"},
				},
			},
		},
	}
	applyNewOrderItemsRemote(payload)

	var items []kdsmodels.KDSItems
	require.NoError(t, db.Where("kds_order_id = ?", order.ID).Find(&items).Error)
	require.Len(t, items, 1)

	var subs []kdsmodels.KDSSubitems
	require.NoError(t, db.Where("kds_item_id = ?", items[0].ID).Find(&subs).Error)
	require.Len(t, subs, 1)
	assert.Equal(t, "medium rare", subs[0].Name)
}

// The unrecognized-reference case: this terminal has no copy of the
// referenced order at all (e.g. the broadcast was lost or arrived on a
// terminal that never had this ticket) - must be a silent no-op, matching
// every other remote handler's reference-not-found behavior, never a panic.
func TestApplyNewOrderItemsRemote_UnknownReferenceIsNoop(t *testing.T) {
	db := setupCloseOrderTestDB(t)

	payload := kdsmodels.KDSNewItemsOnOrder{
		Reference: "no-such-order",
		Items: []kdsmodels.KDSItems{
			{Name: "sprite 300ml", Qty: 1, Status: "OPEN", Screen: kdsmodels.KDSScreens{Name: "bar"}},
		},
	}
	assert.NotPanics(t, func() { applyNewOrderItemsRemote(payload) })

	var count int64
	db.Model(&kdsmodels.KDSItems{}).Count(&count)
	assert.Equal(t, int64(0), count, "nothing should have been created for an order this terminal doesn't have")
}

// Confirmed live: UDP delivery is not exactly-once on this network shape -
// the global broadcast address and the subnet-directed broadcast address
// both reach the same machine, so a receiving terminal's listener gets one
// broadcastNewItemsOnOrder call as two identical packets. Applying the same
// payload twice must create the item once, not twice.
func TestApplyNewOrderItemsRemote_DuplicateDeliveryDoesNotDuplicateItem(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	mustCreateScreen(t, db, "bar", "")
	order := mustCreateOrder(t, db, "ref-dup-delivery")

	payload := kdsmodels.KDSNewItemsOnOrder{
		Reference: "ref-dup-delivery",
		Items: []kdsmodels.KDSItems{
			{Name: "lemonade", Qty: 1, Status: "OPEN", Created: "2026-10-01 12:05:49", Screen: kdsmodels.KDSScreens{Name: "bar"}},
		},
	}
	applyNewOrderItemsRemote(payload)
	applyNewOrderItemsRemote(payload) // the exact same broadcast, delivered twice

	var items []kdsmodels.KDSItems
	require.NoError(t, db.Where("kds_order_id = ? AND name = ?", order.ID, "lemonade").Find(&items).Error)
	assert.Len(t, items, 1, "the same event delivered twice must only create the item once")
}

// A genuinely later, separate round adding the same-named dish again (a
// different Created timestamp, since it was actually stamped at a later,
// real ingestion time) must NOT be mistaken for a duplicate delivery.
func TestApplyNewOrderItemsRemote_SameNameLaterRoundIsNotTreatedAsDuplicate(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	mustCreateScreen(t, db, "bar", "")
	order := mustCreateOrder(t, db, "ref-second-round")

	applyNewOrderItemsRemote(kdsmodels.KDSNewItemsOnOrder{
		Reference: "ref-second-round",
		Items:     []kdsmodels.KDSItems{{Name: "lemonade", Qty: 1, Status: "OPEN", Created: "2026-10-01 12:05:49", Screen: kdsmodels.KDSScreens{Name: "bar"}}},
	})
	applyNewOrderItemsRemote(kdsmodels.KDSNewItemsOnOrder{
		Reference: "ref-second-round",
		Items:     []kdsmodels.KDSItems{{Name: "lemonade", Qty: 1, Status: "OPEN", Created: "2026-10-01 12:19:04", Screen: kdsmodels.KDSScreens{Name: "bar"}}},
	})

	var items []kdsmodels.KDSItems
	require.NoError(t, db.Where("kds_order_id = ? AND name = ?", order.ID, "lemonade").Find(&items).Error)
	assert.Len(t, items, 2, "a later round with a genuinely different Created timestamp is a real second item, not a duplicate")
}

func TestApplyNewOrderItemsRemote_EmptyItemsIsNoop(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	order := mustCreateOrder(t, db, "ref-empty")

	applyNewOrderItemsRemote(kdsmodels.KDSNewItemsOnOrder{Reference: "ref-empty", Items: nil})

	var count int64
	db.Model(&kdsmodels.KDSItems{}).Where("kds_order_id = ?", order.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}
