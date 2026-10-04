package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
)

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

// reconcileSnapshotHandler serves this terminal's own current open-order
// list to a peer's reconcileWithPeer pull, in the same shape
// getAllDatabaseOrdersKDS already produces for the local frontend's own
// initial board load - Occurrence is computed fresh per item before
// marshaling (gorm:"-", never persisted) the same way every single-item
// remote-sync path already does before broadcasting, since it's how a
// peer can address "this specific item" without shared auto-increment ids
// (see itemOccurrenceInOrder's own doc comment).
func reconcileSnapshotHandler(w http.ResponseWriter, r *http.Request) {
	orders := getAllDatabaseOrdersKDS()
	for i := range orders {
		for j := range orders[i].Items {
			orders[i].Items[j].Occurrence = itemOccurrenceInOrder(orders[i].Items[j])
		}
	}
	js, err := json.Marshal(orders)
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

	var peerOrders []kdsmodels.KDSOrders
	if err := json.NewDecoder(resp.Body).Decode(&peerOrders); err != nil {
		fmt.Println("reconcile: decode failed: ", err)
		return false
	}

	changed := false
	for _, peerOrder := range peerOrders {
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
