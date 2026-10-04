package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// D02: reconcileWithPeer pulls a peer's own snapshot and catches this
// terminal's copy up on anything the peer already shows as more done -
// never the reverse. Reuses setupCloseOrderTestDB/mustCreateOrder/
// mustCreateItem/mustCreateScreen from closeorder_scope_test.go.

// peerSnapshotServer starts an httptest server that serves a fixed
// response body for GET /reconcilesnapshot, standing in for a real peer's
// reconcileSnapshotHandler without sharing this test's own ENV globals.
func peerSnapshotServer(t *testing.T, orders []kdsmodels.KDSOrders) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		js, err := json.Marshal(orders)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// peerHostFromServerURL strips "http://" so the result matches what a real
// config's Duplicate.Peers entry looks like (a bare host:port, which
// reconcileWithPeer/peerHostPort re-prefixes with "http://" itself).
func peerHostFromServerURL(url string) string {
	return strings.TrimPrefix(url, "http://")
}

func TestReconcileSnapshotHandler_IncludesOccurrence(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	order := mustCreateOrder(t, db, "ref-snapshot")
	screen := mustCreateScreen(t, db, "kitchen", "")
	mustCreateItem(t, db, order.ID, screen.ID, "steak", "OPEN")
	mustCreateItem(t, db, order.ID, screen.ID, "steak", "OPEN")

	req := httptest.NewRequest(http.MethodGet, "/reconcilesnapshot", nil)
	rec := httptest.NewRecorder()
	reconcileSnapshotHandler(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var orders []kdsmodels.KDSOrders
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &orders))
	require.Len(t, orders, 1)
	require.Len(t, orders[0].Items, 2)
	occurrences := []int{orders[0].Items[0].Occurrence, orders[0].Items[1].Occurrence}
	assert.ElementsMatch(t, []int{1, 2}, occurrences, "same-named items must each carry their own distinct occurrence")
}

func TestReconcileWithPeer_CatchesUpMissedCompletion(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	order := mustCreateOrder(t, db, "ref-missed-close")
	screen := mustCreateScreen(t, db, "kitchen", "")
	item := mustCreateItem(t, db, order.ID, screen.ID, "burger", "OPEN")

	peerItem := kdsmodels.KDSItems{Name: "burger", Qty: 1, Status: "COMPLETE", Occurrence: 1}
	peerOrder := kdsmodels.KDSOrders{Reference: "ref-missed-close", Items: []kdsmodels.KDSItems{peerItem}}
	srv := peerSnapshotServer(t, []kdsmodels.KDSOrders{peerOrder})

	changed := reconcileWithPeer(peerHostFromServerURL(srv.URL))
	assert.True(t, changed)

	var got kdsmodels.KDSItems
	require.NoError(t, db.Where("id = ?", item.ID).First(&got).Error)
	assert.Equal(t, "COMPLETE", got.Status, "a completion the peer already has must be caught up locally")
}

func TestReconcileWithPeer_CatchesUpSentToTable(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	order := mustCreateOrder(t, db, "ref-missed-send")
	screen := mustCreateScreen(t, db, "kitchen", "")
	item := mustCreateItem(t, db, order.ID, screen.ID, "burger", "COMPLETE")

	peerItem := kdsmodels.KDSItems{Name: "burger", Qty: 1, Status: "COMPLETE", SentToTable: true, Occurrence: 1}
	peerOrder := kdsmodels.KDSOrders{Reference: "ref-missed-send", Items: []kdsmodels.KDSItems{peerItem}}
	srv := peerSnapshotServer(t, []kdsmodels.KDSOrders{peerOrder})

	changed := reconcileWithPeer(peerHostFromServerURL(srv.URL))
	assert.True(t, changed)

	var got kdsmodels.KDSItems
	require.NoError(t, db.Where("id = ?", item.ID).First(&got).Error)
	assert.True(t, got.SentToTable, "a send-to-table the peer already has must be caught up locally")
}

func TestReconcileWithPeer_NeverRevertsAlreadyCompleteToOpen(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	order := mustCreateOrder(t, db, "ref-never-revert")
	screen := mustCreateScreen(t, db, "kitchen", "")
	item := mustCreateItem(t, db, order.ID, screen.ID, "burger", "COMPLETE")

	// The peer's own copy is stale/behind (still OPEN) - reconciliation
	// must never move this terminal's own, more-progressed state backwards.
	peerItem := kdsmodels.KDSItems{Name: "burger", Qty: 1, Status: "OPEN", Occurrence: 1}
	peerOrder := kdsmodels.KDSOrders{Reference: "ref-never-revert", Items: []kdsmodels.KDSItems{peerItem}}
	srv := peerSnapshotServer(t, []kdsmodels.KDSOrders{peerOrder})

	changed := reconcileWithPeer(peerHostFromServerURL(srv.URL))
	assert.False(t, changed)

	var got kdsmodels.KDSItems
	require.NoError(t, db.Where("id = ?", item.ID).First(&got).Error)
	assert.Equal(t, "COMPLETE", got.Status)
}

func TestReconcileWithPeer_SkipsOrderNotPresentLocally(t *testing.T) {
	setupCloseOrderTestDB(t)
	peerOrder := kdsmodels.KDSOrders{
		Reference: "ref-unknown-here",
		Items:     []kdsmodels.KDSItems{{Name: "burger", Qty: 1, Status: "COMPLETE", Occurrence: 1}},
	}
	srv := peerSnapshotServer(t, []kdsmodels.KDSOrders{peerOrder})

	assert.NotPanics(t, func() {
		changed := reconcileWithPeer(peerHostFromServerURL(srv.URL))
		assert.False(t, changed, "an order this terminal has never seen is new-order sync's job, not reconciliation's")
	})
}

func TestReconcileWithPeer_UnreachablePeerIsNoop(t *testing.T) {
	setupCloseOrderTestDB(t)
	assert.NotPanics(t, func() {
		changed := reconcileWithPeer("127.0.0.1:1")
		assert.False(t, changed)
	})
}
