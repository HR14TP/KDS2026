package main

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Marking every item on an order COMPLETE must never close/finalize the
// order itself - closing is always a deliberate, separate Close Order
// action (the local button, or the sub-order-close NATS message), never an
// automatic side effect of completing the last item. See closeItemsHandler's
// own doc comment in close_order.go. Confirmed live on the real mesh on
// 2026-10-04 at the user's explicit request, after the standalonemode fix
// extended who can call this endpoint - this guards that the fix didn't
// (and never should) change this unrelated guarantee.

func TestCloseItemsHandler_CompletingOnlyItemNeverClosesOrder(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	setFlags(t, true, false)
	screen := mustCreateScreen(t, db, "kitchen", "")
	order := mustCreateOrder(t, db, "ref-no-autoclose-1")
	item := mustCreateItem(t, db, order.ID, screen.ID, "steak", "OPEN")

	req := withURLVars(httptest.NewRequest(http.MethodGet, "/closeitem/"+strconv.Itoa(item.ID), nil), map[string]string{"id": strconv.Itoa(item.ID)})
	rr := httptest.NewRecorder()
	closeItemsHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var reloadedItem kdsmodels.KDSItems
	require.NoError(t, db.Where("id = ?", item.ID).First(&reloadedItem).Error)
	assert.Equal(t, "COMPLETE", reloadedItem.Status, "the item itself must still complete")
	assertOrderStatus(t, db, order.ID, "OPEN")
}

func TestCloseItemsHandler_CompletingAllItemsNeverClosesOrder(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	setFlags(t, true, false)
	screen := mustCreateScreen(t, db, "kitchen", "")
	order := mustCreateOrder(t, db, "ref-no-autoclose-2")
	itemA := mustCreateItem(t, db, order.ID, screen.ID, "steak", "OPEN")
	itemB := mustCreateItem(t, db, order.ID, screen.ID, "fries", "OPEN")

	for _, item := range []kdsmodels.KDSItems{itemA, itemB} {
		req := withURLVars(httptest.NewRequest(http.MethodGet, "/closeitem/"+strconv.Itoa(item.ID), nil), map[string]string{"id": strconv.Itoa(item.ID)})
		rr := httptest.NewRecorder()
		closeItemsHandler(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
	}

	var reloadedA, reloadedB kdsmodels.KDSItems
	require.NoError(t, db.Where("id = ?", itemA.ID).First(&reloadedA).Error)
	require.NoError(t, db.Where("id = ?", itemB.ID).First(&reloadedB).Error)
	assert.Equal(t, "COMPLETE", reloadedA.Status)
	assert.Equal(t, "COMPLETE", reloadedB.Status)
	assertOrderStatus(t, db, order.ID, "OPEN")
}

func TestCloseItemsHandler_CompletingAllItemsOnStandaloneTerminalNeverClosesOrder(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	setFlags(t, false, true)
	screen := mustCreateScreen(t, db, "bar", "")
	order := mustCreateOrder(t, db, "ref-no-autoclose-3")
	item := mustCreateItem(t, db, order.ID, screen.ID, "beer", "OPEN")

	req := withURLVars(httptest.NewRequest(http.MethodGet, "/closeitem/"+strconv.Itoa(item.ID), nil), map[string]string{"id": strconv.Itoa(item.ID)})
	rr := httptest.NewRecorder()
	closeItemsHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assertOrderStatus(t, db, order.ID, "OPEN")
}
