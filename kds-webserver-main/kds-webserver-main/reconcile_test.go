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
// peerScreen stands in for that peer's own Client.Screen - see
// KDSReconcileSnapshot's own doc comment for why the order-level catch-up
// needs it.
func peerSnapshotServer(t *testing.T, peerScreen string, orders []kdsmodels.KDSOrders) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		js, err := json.Marshal(KDSReconcileSnapshot{Screen: peerScreen, Orders: orders})
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
	ENV.Config.Client.Screen = "kitchen"
	order := mustCreateOrder(t, db, "ref-snapshot")
	screen := mustCreateScreen(t, db, "kitchen", "")
	mustCreateItem(t, db, order.ID, screen.ID, "steak", "OPEN")
	mustCreateItem(t, db, order.ID, screen.ID, "steak", "OPEN")

	req := httptest.NewRequest(http.MethodGet, "/reconcilesnapshot", nil)
	rec := httptest.NewRecorder()
	reconcileSnapshotHandler(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var snapshot KDSReconcileSnapshot
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &snapshot))
	assert.Equal(t, "kitchen", snapshot.Screen, "snapshot must carry this terminal's own screen name")
	require.Len(t, snapshot.Orders, 1)
	require.Len(t, snapshot.Orders[0].Items, 2)
	occurrences := []int{snapshot.Orders[0].Items[0].Occurrence, snapshot.Orders[0].Items[1].Occurrence}
	assert.ElementsMatch(t, []int{1, 2}, occurrences, "same-named items must each carry their own distinct occurrence")
}

// A terminal that already closed an order must still include it in its own
// snapshot - that's the only way a peer who missed the close broadcast can
// ever find out it happened (see TestReconcileWithPeer_CatchesUpMissedClose).
// getAllDatabaseOrdersKDS (OPEN only, what the live board renders) would
// drop it; getTodaysOrdersForReconcile must not.
func TestReconcileSnapshotHandler_IncludesAlreadyClosedOrders(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	ENV.Config.Client.Screen = "kitchen"
	screen := mustCreateScreen(t, db, "kitchen", "")
	order := mustCreateOrder(t, db, "ref-already-closed")
	mustCreateItem(t, db, order.ID, screen.ID, "burger", "COMPLETE")
	closeOrderForPassScreen(order.ID, "kitchen", true)

	req := httptest.NewRequest(http.MethodGet, "/reconcilesnapshot", nil)
	rec := httptest.NewRecorder()
	reconcileSnapshotHandler(rec, req)

	var snapshot KDSReconcileSnapshot
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &snapshot))
	require.Len(t, snapshot.Orders, 1, "a closed order must still appear in this terminal's own reconcile snapshot")
	assert.Equal(t, "COMPLETE", snapshot.Orders[0].Status)
}

func TestReconcileWithPeer_CatchesUpMissedCompletion(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	order := mustCreateOrder(t, db, "ref-missed-close")
	screen := mustCreateScreen(t, db, "kitchen", "")
	item := mustCreateItem(t, db, order.ID, screen.ID, "burger", "OPEN")

	peerItem := kdsmodels.KDSItems{Name: "burger", Qty: 1, Status: "COMPLETE", Occurrence: 1}
	peerOrder := kdsmodels.KDSOrders{Reference: "ref-missed-close", Items: []kdsmodels.KDSItems{peerItem}}
	srv := peerSnapshotServer(t, "kitchen", []kdsmodels.KDSOrders{peerOrder})

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
	srv := peerSnapshotServer(t, "kitchen", []kdsmodels.KDSOrders{peerOrder})

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
	srv := peerSnapshotServer(t, "kitchen", []kdsmodels.KDSOrders{peerOrder})

	changed := reconcileWithPeer(peerHostFromServerURL(srv.URL))
	assert.False(t, changed)

	var got kdsmodels.KDSItems
	require.NoError(t, db.Where("id = ?", item.ID).First(&got).Error)
	assert.Equal(t, "COMPLETE", got.Status)
}

// The core fix this test guards: a dropped closeorderscreenremote message
// (every delivery attempt failing at once - the live symptom this was
// written for) must not strand an order open forever. A sub-screen
// genuinely deferred to the pass terminal that closed it must catch up
// within one reconcile cycle.
func TestReconcileWithPeer_CatchesUpMissedClose(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	// pnp defers to "kitchen" as its pass terminal - same relationship
	// isDeferredToPassTerminal checks in the real-time cascade.
	order := mustCreateOrder(t, db, "ref-missed-order-close")
	screen := mustCreateScreen(t, db, "pnp", "kitchen")
	mustCreateItem(t, db, order.ID, screen.ID, "burger", "COMPLETE")

	// The peer (kitchen) already closed its own copy of this order, but
	// the real-time broadcast never arrived here.
	peerOrder := kdsmodels.KDSOrders{Reference: "ref-missed-order-close", Status: "COMPLETE"}
	srv := peerSnapshotServer(t, "kitchen", []kdsmodels.KDSOrders{peerOrder})

	changed := reconcileWithPeer(peerHostFromServerURL(srv.URL))
	assert.True(t, changed)

	var got kdsmodels.KDSOrders
	require.NoError(t, db.Where("id = ?", order.ID).First(&got).Error)
	assert.Equal(t, "COMPLETE", got.Status, "a close the deferred-to pass terminal already applied must be caught up locally")
}

// The safety counterpart to the test above: a standalone terminal (or any
// screen not actually deferred to whichever terminal sent this snapshot)
// must never have its own order closed just because a same-Reference row
// happens to be complete elsewhere. This is the exact live incident
// closeOrderForPassScreen's own doc comment describes for the real-time
// path, reproduced here for the reconcile path.
func TestReconcileWithPeer_NeverClosesWhenNotDeferredToSender(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	// bar has no assigned_pass_terminal at all - it bumps its own orders.
	mustCreateScreen(t, db, "bar", "")
	order := mustCreateOrder(t, db, "ref-shared-not-deferred")
	screen := mustCreateScreen(t, db, "bar", "")
	mustCreateItem(t, db, order.ID, screen.ID, "beer", "COMPLETE")

	// "kitchen" (not bar's pass terminal) happens to have already closed
	// its own, unrelated row sharing this same Reference.
	peerOrder := kdsmodels.KDSOrders{Reference: "ref-shared-not-deferred", Status: "COMPLETE"}
	srv := peerSnapshotServer(t, "kitchen", []kdsmodels.KDSOrders{peerOrder})

	changed := reconcileWithPeer(peerHostFromServerURL(srv.URL))
	assert.False(t, changed, "bar is not deferred to kitchen - its own order must never be closed by this")

	var got kdsmodels.KDSOrders
	require.NoError(t, db.Where("id = ?", order.ID).First(&got).Error)
	assert.Equal(t, "OPEN", got.Status)
}

func TestReconcileWithPeer_SkipsOrderNotPresentLocally(t *testing.T) {
	setupCloseOrderTestDB(t)
	peerOrder := kdsmodels.KDSOrders{
		Reference: "ref-unknown-here",
		Items:     []kdsmodels.KDSItems{{Name: "burger", Qty: 1, Status: "COMPLETE", Occurrence: 1}},
	}
	srv := peerSnapshotServer(t, "kitchen", []kdsmodels.KDSOrders{peerOrder})

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
