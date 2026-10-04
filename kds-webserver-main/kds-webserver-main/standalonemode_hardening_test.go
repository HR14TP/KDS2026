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

// Standalone-mode hardening: K02 gated closeItemsHandler/sendItemToTableHandler
// purely on Flags.Passmode, which silently swept in standalone terminals too -
// a standalone screen "closes its own items without needing a pass-terminal
// middleman" (see the doc comment on KDS.nats.close_order in kds.js), which is
// a different, independently legitimate category from the "inadvertent
// non-pass terminal" case K02 exists to block. closeItemsHandler now also
// allows Standalonemode; sendItemToTableHandler intentionally does not (see
// its own doc comment: that step is pass-terminal-only by design).

func setFlags(t *testing.T, passmode bool, standalone bool) {
	t.Helper()
	ENV.Config = &kdsmodels.KDSConfig{}
	ENV.Config.Flags.Passmode = passmode
	ENV.Config.Flags.Standalonemode = standalone
}

func TestCloseItemsHandler_AllowsOnStandaloneTerminal(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	setFlags(t, false, true)
	screen := mustCreateScreen(t, db, "bar", "")
	order := mustCreateOrder(t, db, "ref-standalone-1")
	item := mustCreateItem(t, db, order.ID, screen.ID, "beer", "OPEN")

	req := withURLVars(httptest.NewRequest(http.MethodGet, "/closeitem/"+strconv.Itoa(item.ID), nil), map[string]string{"id": strconv.Itoa(item.ID)})
	rr := httptest.NewRecorder()
	closeItemsHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "a standalone terminal must be able to mark its own items complete")
	var reloaded kdsmodels.KDSItems
	require.NoError(t, db.Where("id = ?", item.ID).First(&reloaded).Error)
	assert.Equal(t, "COMPLETE", reloaded.Status)
}

func TestCloseItemsHandler_StillRejectsPlainNonPassNonStandaloneTerminal(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	setFlags(t, false, false)
	screen := mustCreateScreen(t, db, "kitchen", "")
	order := mustCreateOrder(t, db, "ref-standalone-2")
	item := mustCreateItem(t, db, order.ID, screen.ID, "steak", "OPEN")

	req := withURLVars(httptest.NewRequest(http.MethodGet, "/closeitem/"+strconv.Itoa(item.ID), nil), map[string]string{"id": strconv.Itoa(item.ID)})
	rr := httptest.NewRecorder()
	closeItemsHandler(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code, "a plain section terminal (neither pass nor standalone) must still be blocked - K02's original case")
}

func TestSendItemToTableHandler_StillRejectsOnStandaloneTerminal(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	setFlags(t, false, true)
	screen := mustCreateScreen(t, db, "bar", "")
	order := mustCreateOrder(t, db, "ref-standalone-3")
	item := mustCreateItem(t, db, order.ID, screen.ID, "beer", "COMPLETE")

	req := withURLVars(httptest.NewRequest(http.MethodGet, "/senditemtotable/"+strconv.Itoa(item.ID), nil), map[string]string{"id": strconv.Itoa(item.ID)})
	rr := httptest.NewRecorder()
	sendItemToTableHandler(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code, "send-to-table stays pass-terminal-only by design even for standalone terminals")
}

func TestNormalizeStandaloneMode_ForcesPassmodeOffAndClearsPassTerminal(t *testing.T) {
	c := &kdsmodels.KDSConfig{}
	c.Flags.Standalonemode = true
	c.Flags.Passmode = true
	c.Client.Passterminal = "headchef"

	normalizeStandaloneMode(c)

	assert.False(t, c.Flags.Passmode, "standalone must force passmode off even if the file on disk says otherwise")
	assert.Equal(t, "", c.Client.Passterminal, "standalone must clear any stray assigned pass terminal even if the file on disk says otherwise")
}

func TestNormalizeStandaloneMode_LeavesNonStandaloneConfigUntouched(t *testing.T) {
	c := &kdsmodels.KDSConfig{}
	c.Flags.Standalonemode = false
	c.Flags.Passmode = true
	c.Client.Passterminal = "headchef"

	normalizeStandaloneMode(c)

	assert.True(t, c.Flags.Passmode)
	assert.Equal(t, "headchef", c.Client.Passterminal)
}
