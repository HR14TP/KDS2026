package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// D01: a table whose order is fully sent (every real item COMPLETE and
// physically delivered to the table) but never explicitly bumped (order
// Status still OPEN - nothing but an actual Close Order press ever flips
// that, see completeOrder's own doc comment) must have its next ticket
// carrying new items start a genuinely new order, not get silently merged
// into the already-finished one. Reuses setupCloseOrderTestDB/mustCreateItem
// from closeorder_scope_test.go and setPassmode from
// k02_passmode_guard_test.go - same package, same patterns.

func TestOrderIsFullySentButOpen_AllItemsCompleteAndSent(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	order := mustCreateOrder(t, db, "ref-fully-sent")
	screen := mustCreateScreen(t, db, "kitchen", "")
	item := mustCreateItem(t, db, order.ID, screen.ID, "burger", "COMPLETE")
	item.SentToTable = true
	require.NoError(t, db.Save(&item).Error)

	assert.True(t, orderIsFullySentButOpen(order.ID))
}

func TestOrderIsFullySentButOpen_OneItemStillOpen(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	order := mustCreateOrder(t, db, "ref-still-open")
	screen := mustCreateScreen(t, db, "kitchen", "")
	sent := mustCreateItem(t, db, order.ID, screen.ID, "burger", "COMPLETE")
	sent.SentToTable = true
	require.NoError(t, db.Save(&sent).Error)
	mustCreateItem(t, db, order.ID, screen.ID, "fries", "OPEN")

	assert.False(t, orderIsFullySentButOpen(order.ID))
}

func TestOrderIsFullySentButOpen_CompleteButNotYetSentToTable(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	order := mustCreateOrder(t, db, "ref-complete-not-sent")
	screen := mustCreateScreen(t, db, "kitchen", "")
	mustCreateItem(t, db, order.ID, screen.ID, "burger", "COMPLETE") // SentToTable left false

	assert.False(t, orderIsFullySentButOpen(order.ID), "COMPLETE in the kitchen isn't the same as physically delivered")
}

func TestOrderIsFullySentButOpen_OnlyVoidItemsIsNotFullySent(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	order := mustCreateOrder(t, db, "ref-void-only")
	screen := mustCreateScreen(t, db, "kitchen", "")
	mustCreateItem(t, db, order.ID, screen.ID, "burger", "VOID")

	assert.False(t, orderIsFullySentButOpen(order.ID), "a void-only order was never actually delivered")
}

func TestOrderIsFullySentButOpen_NoItemsAtAll(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	order := mustCreateOrder(t, db, "ref-empty")
	_ = db

	assert.False(t, orderIsFullySentButOpen(order.ID), "a brand new order with nothing on it yet is not 'done'")
}

func postSallyOrder(t *testing.T, sallyOrder kdsmodels.SallyOrder) {
	t.Helper()
	body, err := json.Marshal(sallyOrder)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/order", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	ProcessSallyOrder(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

func TestProcessSallyOrder_FullySentOrderStartsFreshOnNextTicketWithItems(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	setPassmode(t, false)

	// Round 1: a burger, fully completed and sent - but never bumped
	// (Status stays OPEN, matching real life: nothing but Close Order
	// itself ever flips that).
	postSallyOrder(t, kdsmodels.SallyOrder{
		Id:            "D01TEST-001",
		ProductCenter: "kitchen",
		TableName:     "Table 9",
		Items:         []kdsmodels.SallyMainItems{{Name: "burger", Qty: 1}},
	})
	var firstOrder kdsmodels.KDSOrders
	require.NoError(t, db.Where("reference = ?", "D01TEST-001").First(&firstOrder).Error)
	var firstItem kdsmodels.KDSItems
	require.NoError(t, db.Where("kds_order_id = ?", firstOrder.ID).First(&firstItem).Error)
	firstItem.Status = "COMPLETE"
	firstItem.SentToTable = true
	require.NoError(t, db.Save(&firstItem).Error)
	require.True(t, orderIsFullySentButOpen(firstOrder.ID))

	// Round 2: the POS reuses the same reference (e.g. the table's next
	// sitting) with a new item.
	postSallyOrder(t, kdsmodels.SallyOrder{
		Id:            "D01TEST-001",
		ProductCenter: "kitchen",
		TableName:     "Table 9",
		Items:         []kdsmodels.SallyMainItems{{Name: "pizza", Qty: 1}},
	})

	var orders []kdsmodels.KDSOrders
	require.NoError(t, db.Where("reference = ?", "D01TEST-001").Find(&orders).Error)
	require.Len(t, orders, 2, "the fully-sent order must be left alone and a genuinely new order created")

	var firstOrderItems, secondOrderItems []kdsmodels.KDSItems
	for _, o := range orders {
		if o.ID == firstOrder.ID {
			db.Where("kds_order_id = ?", o.ID).Find(&firstOrderItems)
		} else {
			db.Where("kds_order_id = ?", o.ID).Find(&secondOrderItems)
		}
	}
	require.Len(t, firstOrderItems, 1, "the original order must not have gained the new ticket's item")
	assert.Equal(t, "burger", firstOrderItems[0].Name)
	require.Len(t, secondOrderItems, 1)
	assert.Equal(t, "pizza", secondOrderItems[0].Name, "the new item must land on the new order, not the old one")
}

func TestProcessSallyOrder_StillOpenOrderAppendsNormally(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	setPassmode(t, false)

	postSallyOrder(t, kdsmodels.SallyOrder{
		Id:            "D01TEST-002",
		ProductCenter: "kitchen",
		TableName:     "Table 4",
		Items:         []kdsmodels.SallyMainItems{{Name: "burger", Qty: 1}},
	})
	// Deliberately NOT marking the burger complete/sent - this order is
	// genuinely still in progress.

	postSallyOrder(t, kdsmodels.SallyOrder{
		Id:            "D01TEST-002",
		ProductCenter: "kitchen",
		TableName:     "Table 4",
		Items:         []kdsmodels.SallyMainItems{{Name: "burger", Qty: 1}, {Name: "fries", Qty: 1}},
	})

	var orders []kdsmodels.KDSOrders
	require.NoError(t, db.Where("reference = ?", "D01TEST-002").Find(&orders).Error)
	require.Len(t, orders, 1, "a still-open order must keep receiving its own round 2 items, not fork into a second order")

	var items []kdsmodels.KDSItems
	require.NoError(t, db.Where("kds_order_id = ?", orders[0].ID).Find(&items).Error)
	assert.Len(t, items, 2, "fries must be appended to the same still-open order")
}
