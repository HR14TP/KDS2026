package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
)

// migrateProductionColumns adds the "in production" columns to kds_items on
// already-deployed SQLite databases. ALTER TABLE errors (column already
// exists) are ignored, matching the convention used by kds-poller's migrate().
func migrateProductionColumns() {
	ENV.DB.Exec("ALTER TABLE kds_items ADD in_production BOOLEAN DEFAULT 0;")
	ENV.DB.Exec("ALTER TABLE kds_items ADD production_started_at DATETIME;")
	ENV.DB.Exec("ALTER TABLE kds_items ADD production_ended_at DATETIME;")
	ENV.DB.Exec("ALTER TABLE kds_orders ADD transfer_note TEXT;")
	ENV.DB.Exec("ALTER TABLE kds_items ADD pushed_to_ods BOOLEAN DEFAULT 0;")
	ENV.DB.Exec("ALTER TABLE kds_items ADD pushed_to_ods_at DATETIME;")
	ENV.DB.Exec("ALTER TABLE kds_items ADD sent_to_table BOOLEAN DEFAULT 0;")
	ENV.DB.Exec("ALTER TABLE kds_items ADD sent_to_table_at DATETIME;")
	ENV.DB.Exec("ALTER TABLE kds_screens ADD assigned_pass_terminal TEXT;")
	ENV.DB.Exec("CREATE TABLE IF NOT EXISTS kds_pass_terminals (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, last_seen DATETIME);")
}

// migrateOrphanedBlankScreenItems runs once at boot (see main.go) as a
// repair, not an ongoing behavior: LoadOrCreateScreen/getScreenSally no
// longer create blank-named screens going forward, and closeOrderForPassScreen/
// closeOrderScreenDatabase both now sweep any item with no real screen of
// its own whenever that item's own order is next closed - but an order
// nobody happens to press Close Order on again would otherwise stay stuck
// forever even with that in place, since neither one runs proactively.
// This finds every such item across the whole database and completes it
// directly (same fields closeItemsHandler itself sets), then re-checks
// each affected order via completeOrder in case that was its last blocker
// - so upgrading to this build alone heals every existing stuck order,
// with nobody needing to notice which ones were affected or manually
// close each one.
//
// Matches an item with NO real screen of its own three ways: kds_screen_id
// pointing at a screen row whose name is blank (the original case this
// existed for), kds_screen_id pointing at nothing at all (0/NULL - never
// assigned one in the first place), or kds_screen_id pointing at a screen
// row that no longer exists (deleted since). A plain INNER JOIN only ever
// caught the first case - the other two have no matching kds_screens row
// to join to at all, so they silently passed through untouched and could
// permanently block their order's completeOrder check forever, with
// nothing (not even a pass terminal's force-close) able to reach them.
//
// Also reachable on demand via repairBrokenOrdersHandler (the Client
// Config "Repair Broken Orders" button) for anyone who doesn't want to
// wait for a restart, or wants to confirm nothing is currently stuck.
// Returns how many items it repaired, so both the boot log and that
// button's response can say something more useful than "done".
func migrateOrphanedBlankScreenItems() int {
	var items []kdsmodels.KDSItems
	ENV.DB.Raw(`
		SELECT ki.* FROM kds_items ki
		LEFT JOIN kds_screens ks ON ki.kds_screen_id = ks.id
		WHERE (ks.id IS NULL OR ks.name = '') AND ki.status = 'OPEN'
	`).Scan(&items)
	if len(items) == 0 {
		return 0
	}
	fmt.Println("Repairing", len(items), "item(s) orphaned on a blank-named screen")
	affectedOrders := map[int]bool{}
	for i := range items {
		item := items[i]
		item.Status = "COMPLETE"
		prims := dif2times(fixDateTime(item.Created))
		item.PrepSecs = int(math.Round(prims[1]))
		item.PrepMins = int(math.Round(prims[0]))
		item.Created = fixDateTime(item.Created)
		if item.InProduction {
			now := getTimeSQL()
			item.InProduction = false
			item.ProductionEndedAt = &now
		}
		now := getTimeSQL()
		item.PushedToOds = true
		item.PushedToOdsAt = &now
		ENV.DB.Save(&item)
		ENV.DB.Where("kds_item_id = ?", item.ID).Find(&item.SubItems)
		for j := range item.SubItems {
			item.SubItems[j].Status = "COMPLETE"
			ENV.DB.Save(&item.SubItems[j])
		}
		affectedOrders[item.KDSOrderID] = true
	}
	for orderid := range affectedOrders {
		completeOrder(orderid)
	}
	return len(items)
}

// repairBrokenOrdersHandler backs the Client Config screen's "Repair
// Broken Orders" button (GET http://127.0.0.1:2020/repairbrokenorders) -
// an on-demand trigger for migrateOrphanedBlankScreenItems, for anyone who
// doesn't want to wait for the next restart. Responds with how many items
// it repaired (0 is a perfectly normal, healthy answer) so the button can
// show a real result instead of a bare "done".
func repairBrokenOrdersHandler(w http.ResponseWriter, r *http.Request) {
	repaired := migrateOrphanedBlankScreenItems()
	notifyKDSOrders()
	js, _ := json.Marshal(map[string]int{"repaired": repaired})
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

// startProductionHandler marks a single item as "in production" on this
// terminal, then fans the change out to every other KDS terminal that has
// this order visible (screen duplication mirrors + the configured peer list).
func startProductionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	itemid := vars["id"]
	item := kdsmodels.KDSItems{}
	order := kdsmodels.KDSOrders{}
	ENV.DB.Where("id = ?", itemid).First(&item)
	if item.ID > 0 {
		ENV.DB.Where("id = ?", item.KDSOrderID).First(&order)
		if order.ID > 0 && item.Status == "OPEN" {
			now := getTimeSQL()
			item.InProduction = true
			item.ProductionStartedAt = &now
			item.ProductionEndedAt = nil
			ENV.DB.Save(&item)

			item.Reference = order.Reference
			item.Occurrence = itemOccurrenceInOrder(item)
			itemjs, _ := json.Marshal(&item)
			sendIfNeeded("DUPSTARTPRODUCTION", itemjs)
			sendToPeers("startproductionremote", itemjs)
			broadcastUDP("startproductionremote", itemjs)
		}
	}
	w.Write([]byte("OK"))
	notifyKDSOrders()
}

// clearProductionHandler removes the "in production" marker from an item.
func clearProductionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	itemid := vars["id"]
	item := kdsmodels.KDSItems{}
	order := kdsmodels.KDSOrders{}
	ENV.DB.Where("id = ?", itemid).First(&item)
	if item.ID > 0 {
		ENV.DB.Where("id = ?", item.KDSOrderID).First(&order)
		if order.ID > 0 {
			now := getTimeSQL()
			item.InProduction = false
			item.ProductionEndedAt = &now
			ENV.DB.Save(&item)

			item.Reference = order.Reference
			item.Occurrence = itemOccurrenceInOrder(item)
			itemjs, _ := json.Marshal(&item)
			sendIfNeeded("DUPCLEARPRODUCTION", itemjs)
			sendToPeers("clearproductionremote", itemjs)
			broadcastUDP("clearproductionremote", itemjs)
		}
	}
	w.Write([]byte("OK"))
	notifyKDSOrders()
}

// startProductionRemoteHandler / clearProductionRemoteHandler receive a
// production-marker change from another terminal. Each terminal has its own
// auto-increment IDs and its own SQLite copy of the order, so the incoming
// item is matched the same way the existing close/open remote handlers do:
// by order Reference + item Name + Qty.
func startProductionRemoteHandler(w http.ResponseWriter, r *http.Request) {
	item := kdsmodels.KDSItems{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&item)
	if err != nil {
		fmt.Println(err)
	}
	applyStartProductionRemote(item)
	w.Write([]byte("OK"))
	notifyKDSOrders()
}

// applyStartProductionRemote marks an item "in production" on this
// terminal's own copy of an order, matched by order Reference + item Name +
// Qty + Occurrence (the item's rank among same-Name+Qty siblings, since an
// order can have more than one identical item and every terminal has its
// own auto-increment ids) - a no-op if this terminal doesn't have that item.
// Shared by the HTTP remote endpoint and the UDP discovery listener.
func applyStartProductionRemote(item kdsmodels.KDSItems) {
	order := getDatabaseOrderFromReference(item.Reference)
	if order.ID > 0 {
		existing := getDatabaseItemByOccurrence(order.ID, item.Name, item.Qty, item.Occurrence)
		if existing.ID > 0 {
			now := getTimeSQL()
			existing.InProduction = true
			existing.ProductionStartedAt = &now
			existing.ProductionEndedAt = nil
			ENV.DB.Save(&existing)
		}
	}
}

func clearProductionRemoteHandler(w http.ResponseWriter, r *http.Request) {
	item := kdsmodels.KDSItems{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&item)
	if err != nil {
		fmt.Println(err)
	}
	applyClearProductionRemote(item)
	w.Write([]byte("OK"))
	notifyKDSOrders()
}

// applyClearProductionRemote clears the "in production" marker on this
// terminal's own copy of an order, matched by Occurrence like
// applyStartProductionRemote. Shared by the HTTP remote endpoint and the UDP
// discovery listener.
func applyClearProductionRemote(item kdsmodels.KDSItems) {
	order := getDatabaseOrderFromReference(item.Reference)
	if order.ID > 0 {
		existing := getDatabaseItemByOccurrence(order.ID, item.Name, item.Qty, item.Occurrence)
		if existing.ID > 0 {
			now := getTimeSQL()
			existing.InProduction = false
			existing.ProductionEndedAt = &now
			ENV.DB.Save(&existing)
		}
	}
}

// sendItemToTableHandler marks a single completed item as physically sent to
// the table, then fans the change out to every other KDS terminal the same
// way startProductionHandler does. Only meaningful for an item that's
// already COMPLETE - the frontend only ever shows the button in that state,
// but the status check here is the actual guard against it firing on
// anything else.
func sendItemToTableHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	itemid := vars["id"]
	item := kdsmodels.KDSItems{}
	order := kdsmodels.KDSOrders{}
	ENV.DB.Where("id = ?", itemid).First(&item)
	if item.ID > 0 {
		ENV.DB.Where("id = ?", item.KDSOrderID).First(&order)
		if order.ID > 0 && item.Status == "COMPLETE" {
			now := getTimeSQL()
			item.SentToTable = true
			item.SentToTableAt = &now
			ENV.DB.Save(&item)

			item.Reference = order.Reference
			item.Occurrence = itemOccurrenceInOrder(item)
			itemjs, _ := json.Marshal(&item)
			sendIfNeeded("DUPSENDTOTABLE", itemjs)
			sendToPeers("sendtotableremote", itemjs)
			broadcastUDP("sendtotableremote", itemjs)
		}
	}
	w.Write([]byte("OK"))
	notifyKDSOrders()
}

// sendToTableRemoteHandler receives a sent-to-table change from another
// terminal over HTTP (Duplicate.Remote / Duplicate.Peers).
func sendToTableRemoteHandler(w http.ResponseWriter, r *http.Request) {
	item := kdsmodels.KDSItems{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&item)
	if err != nil {
		fmt.Println(err)
	}
	applySendToTableRemote(item)
	w.Write([]byte("OK"))
	notifyKDSOrders()
}

// applySendToTableRemote marks an item sent-to-table on this terminal's own
// copy of an order, matched by Occurrence like applyStartProductionRemote.
// Shared by the HTTP remote endpoint and the UDP discovery listener.
func applySendToTableRemote(item kdsmodels.KDSItems) {
	order := getDatabaseOrderFromReference(item.Reference)
	if order.ID > 0 {
		existing := getDatabaseItemByOccurrence(order.ID, item.Name, item.Qty, item.Occurrence)
		if existing.ID > 0 {
			now := getTimeSQL()
			existing.SentToTable = true
			existing.SentToTableAt = &now
			ENV.DB.Save(&existing)
		}
	}
}
