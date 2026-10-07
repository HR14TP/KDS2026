package main

import (
	"testing"

	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Three independent layers now guard the "closed on headchef, never
// cascaded to a linked sub-screen, intermittently" symptom: postToHost
// retries a transient HTTP failure (post_to_host_retry_test.go), a
// fast-follow reconcile fires ~10s after every local close instead of
// waiting on the 2-minute periodic timer (natsCloseOrder, nat_coms.go),
// and the periodic/fast-follow reconcile itself now catches up a missed
// order-close, not just item completion (reconcile_test.go). This file
// guards the property all three layers depend on: applying the same close
// more than once - whether from a retried push, the fast-follow pass, AND
// the periodic pass all landing, or a human pressing Close Order twice -
// must always be safe.

func TestCloseOrderForPassScreen_IdempotentAcrossRepeatedCalls(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	screen := mustCreateScreen(t, db, "kitchen", "")
	order := mustCreateOrder(t, db, "ref-idempotent")
	mustCreateItem(t, db, order.ID, screen.ID, "steak", "OPEN")

	first := closeOrderForPassScreen(order.ID, "kitchen", true)
	require.Equal(t, "COMPLETE", first.Status)
	require.NotNil(t, first.Completed)

	// Simulates the fast-follow reconcile pass (or a retried push that
	// landed after all, or a human double-tapping Close Order) re-applying
	// the exact same close a moment later.
	second := closeOrderForPassScreen(order.ID, "kitchen", true)
	assert.Equal(t, "COMPLETE", second.Status, "re-closing an already-closed order must stay closed, not error or revert")
	assert.NotNil(t, second.Completed, "must still carry a completion timestamp after a redundant close")
}

// The same property, specifically exercised the way it actually happens in
// production: a real-time local close followed by the periodic/fast-follow
// reconcile pass discovering (from a peer's snapshot) that this same order
// is already COMPLETE here too - reconcileWithPeer's own gate
// (localOrder.Status == "OPEN") must mean the second layer never even
// attempts a redundant close once the first one already landed.
func TestReconcileWithPeer_DoesNotReapplyAlreadyLocallyClosedOrder(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	screen := mustCreateScreen(t, db, "pnp", "kitchen")
	order := mustCreateOrder(t, db, "ref-already-closed-locally")
	mustCreateItem(t, db, order.ID, screen.ID, "burger", "COMPLETE")
	closeOrderForPassScreen(order.ID, "kitchen", true)

	var beforeReconcile kdsmodels.KDSOrders
	require.NoError(t, db.Where("id = ?", order.ID).First(&beforeReconcile).Error)
	require.Equal(t, "COMPLETE", beforeReconcile.Status)

	peerOrder := kdsmodels.KDSOrders{Reference: "ref-already-closed-locally", Status: "COMPLETE"}
	srv := peerSnapshotServer(t, "kitchen", []kdsmodels.KDSOrders{peerOrder})

	changed := reconcileWithPeer(peerHostFromServerURL(srv.URL))
	assert.False(t, changed, "already-closed-here must be recognized as nothing new to catch up")

	var got kdsmodels.KDSOrders
	require.NoError(t, db.Where("id = ?", order.ID).First(&got).Error)
	assert.Equal(t, "COMPLETE", got.Status)
}
