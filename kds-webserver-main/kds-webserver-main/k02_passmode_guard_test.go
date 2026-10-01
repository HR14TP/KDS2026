package main

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gorilla/mux"
	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// K02: "Only the Head Chef pass KDS may mark an item Complete and then Sent
// to table." These tests cover the server-side enforcement added to
// closeItemsHandler/sendItemToTableHandler - the frontend's own passmode
// check (KDS.close_item/KDS.send_to_table) is only a UI courtesy and was
// never a real security boundary on its own.

func setPassmode(t *testing.T, passmode bool) {
	t.Helper()
	ENV.Config = &kdsmodels.KDSConfig{}
	ENV.Config.Flags.Passmode = passmode
}

func withURLVars(r *http.Request, vars map[string]string) *http.Request {
	return mux.SetURLVars(r, vars)
}

func TestCloseItemsHandler_RejectsOnNonPassTerminal(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	setPassmode(t, false)
	screen := mustCreateScreen(t, db, "kitchen", "")
	order := mustCreateOrder(t, db, "ref-guard-1")
	item := mustCreateItem(t, db, order.ID, screen.ID, "steak", "OPEN")

	req := withURLVars(httptest.NewRequest(http.MethodGet, "/closeitem/"+strconv.Itoa(item.ID), nil), map[string]string{"id": strconv.Itoa(item.ID)})
	rr := httptest.NewRecorder()
	closeItemsHandler(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	var reloaded kdsmodels.KDSItems
	require.NoError(t, db.Where("id = ?", item.ID).First(&reloaded).Error)
	assert.Equal(t, "OPEN", reloaded.Status, "item must be untouched when the request is rejected")
}

func TestCloseItemsHandler_AllowsOnPassTerminal(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	setPassmode(t, true)
	screen := mustCreateScreen(t, db, "kitchen", "")
	order := mustCreateOrder(t, db, "ref-guard-2")
	item := mustCreateItem(t, db, order.ID, screen.ID, "steak", "OPEN")

	req := withURLVars(httptest.NewRequest(http.MethodGet, "/closeitem/"+strconv.Itoa(item.ID), nil), map[string]string{"id": strconv.Itoa(item.ID)})
	rr := httptest.NewRecorder()
	closeItemsHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var reloaded kdsmodels.KDSItems
	require.NoError(t, db.Where("id = ?", item.ID).First(&reloaded).Error)
	assert.Equal(t, "COMPLETE", reloaded.Status)
}

func TestSendItemToTableHandler_RejectsOnNonPassTerminal(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	setPassmode(t, false)
	screen := mustCreateScreen(t, db, "kitchen", "")
	order := mustCreateOrder(t, db, "ref-guard-3")
	item := mustCreateItem(t, db, order.ID, screen.ID, "steak", "COMPLETE")

	req := withURLVars(httptest.NewRequest(http.MethodGet, "/senditemtotable/"+strconv.Itoa(item.ID), nil), map[string]string{"id": strconv.Itoa(item.ID)})
	rr := httptest.NewRecorder()
	sendItemToTableHandler(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	var reloaded kdsmodels.KDSItems
	require.NoError(t, db.Where("id = ?", item.ID).First(&reloaded).Error)
	assert.False(t, reloaded.SentToTable, "item must be untouched when the request is rejected")
}

func TestSendItemToTableHandler_AllowsOnPassTerminal(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	setPassmode(t, true)
	screen := mustCreateScreen(t, db, "kitchen", "")
	order := mustCreateOrder(t, db, "ref-guard-4")
	item := mustCreateItem(t, db, order.ID, screen.ID, "steak", "COMPLETE")

	req := withURLVars(httptest.NewRequest(http.MethodGet, "/senditemtotable/"+strconv.Itoa(item.ID), nil), map[string]string{"id": strconv.Itoa(item.ID)})
	rr := httptest.NewRecorder()
	sendItemToTableHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var reloaded kdsmodels.KDSItems
	require.NoError(t, db.Where("id = ?", item.ID).First(&reloaded).Error)
	assert.True(t, reloaded.SentToTable)
}

