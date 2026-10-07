package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
)

// KDSReconcileSnapshot is the wire shape served by reconcileSnapshotHandler -
// wraps the orders list with the responding terminal's own screen name, so
// a puller can run the exact same assigned_pass_terminal safety check a
// real-time close already does (isDeferredToPassTerminal) before trusting
// this peer's "already closed" state. Without the screen name, reconcile
// couldn't tell "my own configured pass terminal already closed this" apart
// from "some unrelated terminal that happens to share this Reference closed
// its own, unrelated section" - exactly the standalone-terminal bug
// closeOrderForPassScreen's own doc comment describes for the real-time
// path (a Kitchen pass screen's bump must never touch a Standalone Mode
// Bar's row on the same shared reference).
type KDSReconcileSnapshot struct {
	Screen string                `json:"screen"`
	Orders []kdsmodels.KDSOrders `json:"orders"`
}

// D02: every cross-terminal status change in this codebase (DUPCLOSEITEM,
// DUPOPENITEM, DUPSTARTPRODUCTION, DUPSENDTOTABLE, closeorderscreenremote,
// neworderremote, neworderitemsremote, ...) travels one-way and
// fire-and-forget, over NATS mirrors, HTTP peers, and/or UDP broadcast -
// none of them have delivery guarantees (see
// applyNewOrderItemsRemote's own doc comment: UDP here isn't even
// exactly-once, confirmed live, let alone reliable). If every delivery
// attempt for one specific change happens to fail at once (plausible: an
// HTTP POST timeout and a dropped UDP packet, in the same moment, to the
// same peer), that peer's copy of the affected item is stuck wherever it
// was forever - there is no later event that will ever re-examine that
// specific item again. reconcileWithPeers is a periodic, pull-based
// self-heal for exactly that gap: ask each configured peer for its own
// current snapshot of today's open orders, and catch this terminal's own
// copy up on anything the peer already has that this terminal doesn't -
// deliberately one-directional (never un-completes or un-sends anything
// this terminal already shows, only ever fills in a completion/send this
// terminal missed), so a dropped message can only ever be caught up on
// late, never cause a correct local action to be reverted.
//
// Deliberately NOT symmetric with closing/opening an order itself, or with
// syncing OPEN status, InProduction, or anything else this terminal might
// have legitimately already diverged on for its own reasons - only status
// strictly "more done" than this terminal's own copy (COMPLETE, SentToTable)
// is ever pulled in, using the exact same apply-side functions
// (applyCloseRemoteItem/applySendToTableRemote) the real-time push paths
// already use, so a caught-up item ends up in exactly the state a timely
// delivery would have left it in - no new code path, no new side effects.

// getTodaysOrdersForReconcile returns every one of today's orders
// regardless of Status - unlike getAllDatabaseOrdersKDS (OPEN only, what
// the live board itself renders), a peer pulling this snapshot needs to
// see an order THIS terminal already closed too, so it can recognize
// "already closed here" and catch its own stuck-open copy up (see
// reconcileWithPeer's order-close catch-up below).
func getTodaysOrdersForReconcile() []kdsmodels.KDSOrders {
	orders := []kdsmodels.KDSOrders{}
	ENV.DB.Where("strftime('%d-%m-%Y',created) = strftime('%d-%m-%Y', 'now', 'localtime')" + todayOrDateUnparseable).Find(&orders)
	for i := range orders {
		ENV.DB.Where("kds_order_id = ?", orders[i].ID).Find(&orders[i].Items)
	}
	return orders
}

// reconcileSnapshotHandler serves this terminal's own current snapshot -
// every one of today's orders, open or already closed - to a peer's
// reconcileWithPeer pull. Occurrence is computed fresh per item before
// marshaling (gorm:"-", never persisted) the same way every single-item
// remote-sync path already does before broadcasting, since it's how a
// peer can address "this specific item" without shared auto-increment ids
// (see itemOccurrenceInOrder's own doc comment).
func reconcileSnapshotHandler(w http.ResponseWriter, r *http.Request) {
	orders := getTodaysOrdersForReconcile()
	for i := range orders {
		for j := range orders[i].Items {
			orders[i].Items[j].Occurrence = itemOccurrenceInOrder(orders[i].Items[j])
		}
	}
	snapshot := KDSReconcileSnapshot{Screen: ENV.Config.Client.Screen, Orders: orders}
	js, err := json.Marshal(snapshot)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

// reconcileWithPeers pulls a fresh snapshot from every configured peer
// (same target list sendToPeers already fans out to) and catches this
// terminal's own copy of today's still-OPEN orders up on anything the peer
// already shows as more done.
func reconcileWithPeers() {
	targets := []string{}
	if len(ENV.Config.Duplicate.Remote) > 1 {
		targets = append(targets, ENV.Config.Duplicate.Remote)
	}
	for _, peer := range ENV.Config.Duplicate.Peers {
		if len(peer) > 1 {
			targets = append(targets, peer)
		}
	}
	changed := false
	for _, target := range targets {
		if reconcileWithPeer(target) {
			changed = true
		}
	}
	if changed {
		notifyKDSOrders()
	}
}

// reconcileWithPeer pulls one peer's snapshot and applies any catch-up
// locally. Returns whether anything actually changed, so the caller only
// pays for a board refresh when one is actually warranted. A peer that's
// unreachable, or an order this terminal has no copy of at all (that's
// D06/new-order sync's job, not this pass's), is silently skipped - same
// fail-open philosophy as every other peer-sync path in this codebase.
func reconcileWithPeer(peerHost string) bool {
	url := "http://" + peerHostPort(peerHost) + "/reconcilesnapshot"
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("reconcile: peer unreachable: " + url)
		return false
	}
	defer resp.Body.Close()

	var snapshot KDSReconcileSnapshot
	if err := json.NewDecoder(resp.Body).Decode(&snapshot); err != nil {
		fmt.Println("reconcile: decode failed: ", err)
		return false
	}

	changed := false
	for _, peerOrder := range snapshot.Orders {
		localOrder := getDatabaseOrderFromReference(peerOrder.Reference)
		if localOrder.ID == 0 {
			continue
		}
		for _, peerItem := range peerOrder.Items {
			localItem := getDatabaseItemByOccurrence(localOrder.ID, peerItem.Name, peerItem.Qty, peerItem.Occurrence)
			if localItem.ID == 0 {
				continue
			}
			if peerItem.Status == "COMPLETE" && localItem.Status != "COMPLETE" {
				remoteItem := peerItem
				remoteItem.Reference = peerOrder.Reference
				applyCloseRemoteItem(remoteItem)
				changed = true
			}
			if peerItem.SentToTable && !localItem.SentToTable {
				remoteItem := peerItem
				remoteItem.Reference = peerOrder.Reference
				applySendToTableRemote(remoteItem)
				changed = true
			}
		}
		// Catch up a missed order-level close - the gap this whole file
		// exists for, extended beyond items: gated on the exact same
		// assigned_pass_terminal relationship the real-time cascade
		// requires (isDeferredToPassTerminal), so this only ever fires for
		// a sub-screen genuinely deferred to the pass terminal that closed
		// it, never for a standalone terminal merely sharing a Reference.
		if localOrder.Status == "OPEN" && peerOrder.Status == "COMPLETE" && isDeferredToPassTerminal(snapshot.Screen) {
			closeOrderForPassScreen(localOrder.ID, snapshot.Screen, true)
			changed = true
		}
	}
	return changed
}

// asyncReconcileWithPeers runs reconcileWithPeers on a timer, same idiom as
// asyncCheckConnections/asyncAnnouncePassTerminal (see main.go) - catches up
// any divergence left by a dropped delivery within a few minutes, without
// needing a human to ever notice and press something to fix it.
func asyncReconcileWithPeers() {
	for range time.Tick(time.Minute * 2) {
		reconcileWithPeers()
	}
}
