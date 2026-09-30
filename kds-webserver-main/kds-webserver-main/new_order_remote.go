package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
)

// newOrderRemoteHandler receives a brand-new order from another terminal
// over HTTP (Duplicate.Remote / Duplicate.Peers).
func newOrderRemoteHandler(w http.ResponseWriter, r *http.Request) {
	order := kdsmodels.KDSOrders{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&order)
	if err != nil {
		fmt.Println(err)
	}
	applyNewOrderRemote(order)
	w.Write([]byte("OK"))
	notifyKDSOrders()
}

// applyNewOrderRemote creates a local copy of an order that was originally
// ingested on another terminal, preserving its OrderFortoday (the
// human-facing "Chk N" number shown as "Chk N" on the board) instead of
// computing a fresh one here. Every terminal otherwise assigns its own
// OrderFortoday independently from its own local daily counter
// (ProcessSallyOrder/NewOrderForToday), which is fine as long as only one
// terminal ever shows a given ticket - but a terminal that's purely
// announcing orders (an Order Display Screen) must show the exact same
// number the kitchen terminal that actually took the ticket is showing, not
// its own unrelated count of how many orders it happens to have seen today.
//
// If this terminal already has an order with the same Reference - either it
// ingested the ticket itself via its own poller, or it already applied this
// exact broadcast once (broadcasts are fire-and-forget and can arrive more
// than once) - this only reconciles the order's Created timestamp and
// returns; it never rebuilds the order/items.
//
// Every terminal runs its own poller against the same POS print feed and
// stamps Created from its own local clock the moment it finishes parsing a
// ticket (see kds-poller's engine.go) - there is no single canonical
// timestamp for "when this order arrived". Two terminals racing to ingest
// the same ticket almost always both win locally before either one's
// broadcast arrives, so without this reconciliation each keeps its own
// independently-generated Created forever, and the order tile's elapsed-
// time display (anchored to Created) visibly disagrees between terminals
// for the life of the order. Converging to whichever Created is earliest -
// regardless of which terminal's broadcast happens to arrive first - makes
// every terminal settle on the same value once the exchange completes,
// without touching Status/InProduction or anything else that may have
// already progressed locally.
func applyNewOrderRemote(order kdsmodels.KDSOrders) {
	existing := getDatabaseOrderOnReference(config, order.Reference)
	if existing.ID > 0 {
		reconcileRemoteOrderCreated(existing, order)
		return
	}
	neworder := kdsmodels.KDSOrders{}
	neworder.OrderFortoday = order.OrderFortoday
	neworder.Reference = order.Reference
	neworder.Table = order.Table
	neworder.Waiter = order.Waiter
	neworder.Status = order.Status
	neworder.Profile = order.Profile
	neworder.Created = order.Created
	neworder.PassStatus = order.PassStatus
	neworder.Consumer = order.Consumer
	neworder.Delivery = order.Delivery
	neworder = saveSallyOrder(neworder)

	for _, item := range order.Items {
		// KDSScreenID is a local foreign key, not portable between
		// terminals' own databases - map by the screen's name instead
		// (getScreenSally creates it locally on first sight, same as
		// normal ticket ingestion does).
		screen := getScreenSally(item.Screen.Name)
		newitem := kdsmodels.KDSItems{}
		newitem.KDSOrderID = neworder.ID
		newitem.Name = item.Name
		newitem.Qty = item.Qty
		newitem.QtyRaw = item.QtyRaw
		newitem.Status = item.Status
		newitem.KDSScreenID = screen.ID
		newitem.Created = item.Created
		newitem.IsKiddies = item.IsKiddies
		newitem.PrepMins = item.PrepMins
		newitem.PrepSecs = item.PrepSecs
		newitem.SubOrder = item.SubOrder
		newitem = saveSallyItem(newitem)
		for _, sub := range item.SubItems {
			newsub := kdsmodels.KDSSubitems{}
			newsub.KDSItemID = newitem.ID
			newsub.Name = sub.Name
			newsub.Qty = sub.Qty
			newsub.QtyRaw = sub.QtyRaw
			newsub.Status = sub.Status
			saveSallySubItem(newsub)
		}
	}
}

// reconcileRemoteOrderCreated converges the order's Created, and each of its
// items' Created, to whichever value is earliest between what's already
// stored locally and what this broadcast carries - never any later value,
// so this always settles rather than flapping back and forth as broadcasts
// from both sides arrive. Nothing else on the order or its items is
// touched: Status/InProduction/etc. reflect this terminal's own live state
// and must not be overwritten by a remote snapshot that may be stale by the
// time it arrives.
//
// Items are matched by Name+Qty and rank among same-Name+Qty siblings
// (locally ordered by id ASC, in the broadcast by list position) - the same
// disambiguation itemOccurrenceInOrder/getDatabaseItemByOccurrence already
// rely on elsewhere for two terminals whose copies of an order share the
// same items processed off the same feed in the same order. If the two
// sides' item counts for a given name+qty don't line up (order still
// mid-sync, or has genuinely diverged), the extra incoming items are simply
// skipped rather than guessed at.
func reconcileRemoteOrderCreated(existing kdsmodels.KDSOrders, incoming kdsmodels.KDSOrders) {
	if incoming.Created != "" && incoming.Created < existing.Created {
		existing.Created = incoming.Created
		ENV.DB.Save(&existing)
	}

	localItems := []kdsmodels.KDSItems{}
	ENV.DB.Where("kds_order_id = ?", existing.ID).Order("id ASC").Find(&localItems)

	occurrenceSoFar := map[string]int{}
	for _, incomingItem := range incoming.Items {
		key := incomingItem.Name + "\x00" + strconv.Itoa(incomingItem.Qty)
		occurrenceSoFar[key]++
		occurrence := occurrenceSoFar[key]

		siblingIndex := 0
		for _, localItem := range localItems {
			if localItem.Name != incomingItem.Name || localItem.Qty != incomingItem.Qty {
				continue
			}
			siblingIndex++
			if siblingIndex != occurrence {
				continue
			}
			if incomingItem.Created != "" && incomingItem.Created < localItem.Created {
				localItem.Created = incomingItem.Created
				ENV.DB.Save(&localItem)
			}
			break
		}
	}
}
