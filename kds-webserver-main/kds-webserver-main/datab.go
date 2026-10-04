package main

import (
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/davecgh/go-spew/spew"
	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
)

// todayOrDateUnparseable is appended to every "today's orders only" query in
// this file - see getDatabaseOrders's own comment for why: a malformed
// created value (confirmed live: corrupting one to "" while testing the
// frontend's oldest-first sort fix) makes strftime/DATE return NULL, and
// NULL never equals anything in SQL, so a plain equality check silently
// drops that order out of every query using it - including this terminal's
// own single-order lookup (getDatabaseOrder, used by Close Order's
// confirmation dialog) and recall's lookups, permanently, with no error and
// no visible sign anything is wrong. Failing open (still find it) rather
// than failing closed (hide/lose it) means at worst it's found on a day it
// maybe shouldn't be - recoverable - instead of becoming permanently
// unreachable by anything in the UI.
const todayOrDateUnparseable = " OR strftime('%d-%m-%Y', created) IS NULL"

func getDatabaseOrder(config kdsmodels.KDSConfig, id string) kdsmodels.KDSOrders {
	orders := kdsmodels.KDSOrders{}
	ENV.DB.Where("status = ?", "OPEN").Where("strftime('%d-%m-%Y',created) = strftime('%d-%m-%Y',CURRENT_TIMESTAMP)"+todayOrDateUnparseable).Where("id = ?", id).First(&orders)
	ENV.DB.Where("kds_order_id = ?", orders.ID).Find(&orders.Items)
	for j := range orders.Items {
		ENV.DB.Where("kds_item_id = ?", orders.Items[j].ID).Find(&orders.Items[j].SubItems)
		ENV.DB.Where("id = ?", orders.Items[j].KDSScreenID).Find(&orders.Items[j].Screen)
	}
	return orders
}

func getDatabaseOrderOnReference(config kdsmodels.KDSConfig, id string) kdsmodels.KDSOrders {
	orders := kdsmodels.KDSOrders{}
	ENV.DB.Where("status = ?", "OPEN").Where("strftime('%d-%m-%Y',created) = strftime('%d-%m-%Y',CURRENT_TIMESTAMP)"+todayOrDateUnparseable).Where("reference = ?", id).First(&orders)
	return orders
}

// orderIsFullySentButOpen reports whether every real item on this order has
// already been completed and physically sent to the table (Status ==
// "COMPLETE" and SentToTable == true), even though the order's own Status
// is still "OPEN" - i.e. everything on the ticket is actually done, but
// nobody has pressed Close Order (completeOrder only ever fires as a side
// effect of that explicit action, never from an item reaching
// COMPLETE/SentToTable on its own - see its own doc comment). Mirrors
// kds-poller's own copy of this same check (engine.go's hook into
// ProcessMessage) for the Sally/POS-API ingestion path (ProcessSallyOrder).
// VOID/TRANSFERRED/COURSE items don't count as "real" either way, and at
// least one real item is required - see the poller copy's doc comment for
// why both of those matter.
func orderIsFullySentButOpen(orderID int) bool {
	var items []kdsmodels.KDSItems
	ENV.DB.Where("kds_order_id = ?", orderID).Find(&items)
	realItems := 0
	for _, item := range items {
		if item.Status == "VOID" || item.Status == "TRANSFERRED" || item.Status == "COURSE" {
			continue
		}
		realItems++
		if !(item.Status == "COMPLETE" && item.SentToTable) {
			return false
		}
	}
	return realItems > 0
}

func saveSallyOrder(order kdsmodels.KDSOrders) kdsmodels.KDSOrders {
	if order.ID > 0 {
		ENV.DB.Save(&order)
	} else {
		ENV.DB.Create(&order)
	}
	return order
}

func saveSallyItem(item kdsmodels.KDSItems) kdsmodels.KDSItems {
	if item.ID > 0 {
		transaction := ENV.DB.Save(&item)
		spew.Dump(transaction.Error)
	} else {
		transaction := ENV.DB.Create(&item)
		spew.Dump(transaction.Error)
	}
	//spew.Dump(item)
	return item
}

// NewOrderForToday ... NewOrderForToday
func NewOrderForToday() int {
	var res kdsmodels.ForToday
	ENV.DB.Where("SELECT count(*) as counter from kds_orders where DATE(kds_orders.created) = DATE(CURRENT_TIMESTAMP, 'localtime')").Scan(&res)
	return res.Counter
}

func NewSubOrder(kdsorderid int) int {
	var res kdsmodels.ForToday
	ENV.DB.Where("SELECT max(suborder) as counter from kds_items where kds_order_id = ?", kdsorderid).Scan(&res)
	return res.Counter
}

func saveSallySubItem(subitem kdsmodels.KDSSubitems) kdsmodels.KDSSubitems {
	if subitem.ID > 0 {
		transaction := ENV.DB.Save(&subitem)
		spew.Dump(transaction.Error)
	} else {
		transaction := ENV.DB.Create(&subitem)
		spew.Dump(transaction.Error)
	}
	spew.Dump(subitem)
	return subitem
}

// getScreenSally mirrors kds-poller's LoadOrCreateScreen. An empty
// ProductCenter is deliberately left as an empty Name rather than renamed
// to some fallback like "unassigned" - a pass-mode close (unconditional)
// always sweeps up a blank-named screen's items regardless, and
// closeOrderScreenDatabase also always sweeps one up when its own order is
// closed (see its own comment), so leaving it blank is what keeps that
// sweep able to find it. Renaming it
// to a fixed non-blank string here would silently break that sweep for
// every item landing here from this point on - it would only ever match
// screens whose name is genuinely "", not one deliberately renamed away
// from it.
// Screen lookup is case-insensitive (LOWER(name) = LOWER(?)) so "Kitchen"
// and "kitchen" resolve to the same screen row instead of silently
// creating a case-variant duplicate that a differently-cased terminal
// name could never match against.
func getScreenSally(name string) kdsmodels.KDSScreens {
	var screen kdsmodels.KDSScreens
	if name != "" {
		ENV.DB.Where("LOWER(name) = LOWER(?)", name).First(&screen)
	} else {
		ENV.DB.Where("name = ?", name).First(&screen)
	}
	if screen.ID > 0 {
		return screen
	}
	screen.Name = name
	screen.Port = "hdmi"
	ENV.DB.Create(&screen)
	return screen
}

// getItemSally resolves which of this order's items a Sally (POS) update for
// sallyitem.Name actually refers to, the same "occurrence" way
// getDatabaseItemByOccurrence already does for every other cross-terminal
// item-identity lookup in this codebase (close/production/DUP* handlers) -
// occurrence is the item's 1-based rank, by id ASC (creation order), among
// this order's own not-yet-COMPLETE items sharing that name. Deliberately
// NOT scoped by Qty (unlike getDatabaseItemByOccurrence): that function
// matches an already-fully-formed item snapshot broadcast between
// terminals, where qty is assumed stable, whereas this one is reconciling
// an evolving order against a fresh POS round whose whole point can be a
// qty change on an existing item - scoping by qty here would make a qty
// change invisible as "the same item" and wrongly spawn a duplicate.
//
// Before this, a name-only, unordered match (GORM's First() with no
// ORDER BY) meant two same-named items on one order (e.g. two "Rump Steak
// 300gr", one medium one well done) were indistinguishable: whichever
// instance the query happened to return first silently absorbed every
// update meant for the other, including modifiers attached to a specific
// instance.
//
// occurrence must be computed by the caller by counting this item's rank
// among same-named items already seen earlier in the SAME incoming
// sallyOrder.Items list (see processSallyMainItem) - the Sally wire format
// carries no occurrence of its own, so there's nothing to read it from; it
// has to be derived fresh from the feed's own item ordering each round, the
// same way itemOccurrenceInOrder derives it from local id ordering.
//
// Returns a zero-value KDSItems (ID 0) when occurrence has no matching
// sibling yet - e.g. a genuinely new second instance of the same dish just
// added this round - so the caller's existing `existingitem.ID > 0` check
// correctly falls through to creating a new item instead of reusing a
// wrong one. Unlike getDatabaseItemByOccurrence, deliberately does NOT fall
// back to the first sibling out of range: that fallback exists there so a
// remote status toggle still lands somewhere rather than being silently
// dropped, which does not apply to a create-vs-update decision here.
func getItemSally(orderid int, sallyitem kdsmodels.SallyMainItems, occurrence int) kdsmodels.KDSItems {
	siblings := []kdsmodels.KDSItems{}
	ENV.DB.Where("kds_order_id = ?", orderid).Where("name = ?", sallyitem.Name).Where("status != ?", "COMPLETE").Order("id ASC").Find(&siblings)
	if occurrence >= 1 && occurrence <= len(siblings) {
		return siblings[occurrence-1]
	}
	return kdsmodels.KDSItems{}
}

// getSubItemSally is getItemSally's counterpart for an item's own subitems
// (modifiers) - same occurrence-by-name-rank reasoning, scoped to one
// item's own subitems instead of one order's own items. occurrence must be
// computed by the caller the same way, by counting this subitem's rank
// among same-named subitems already seen earlier in the SAME incoming
// sallyOrder.Items[g].SubItems list for this one item.
func getSubItemSally(itemid int, sallysubitem kdsmodels.SallySubItems, occurrence int) kdsmodels.KDSSubitems {
	siblings := []kdsmodels.KDSSubitems{}
	ENV.DB.Where("kds_item_id = ?", itemid).Where("name = ?", sallysubitem.Name).Where("status != ?", "COMPLETE").Order("id ASC").Find(&siblings)
	if occurrence >= 1 && occurrence <= len(siblings) {
		return siblings[occurrence-1]
	}
	return kdsmodels.KDSSubitems{}
}

func getDatabaseOrderWithoutStatus(config kdsmodels.KDSConfig, id string) kdsmodels.KDSOrders {
	order := kdsmodels.KDSOrders{}
	ENV.DB.Where("id = ?", id).First(&order)
	ENV.DB.Where("kds_order_id = ?", order.ID).Find(&order.Items)
	for j := range order.Items {
		ENV.DB.Where("kds_item_id = ?", order.Items[j].ID).Find(&order.Items[j].SubItems)
		ENV.DB.Where("id = ?", order.Items[j].KDSScreenID).Find(&order.Items[j].Screen)
	}
	return order
}

func getCompletedOrdersScreen(screen string) []kdsmodels.KDSOrders {
	kdsscreen := kdsmodels.KDSScreens{}
	ENV.DB.Where("LOWER(name) = LOWER(?)", screen).Find(&kdsscreen)
	if kdsscreen.ID > 0 {
		orders := []kdsmodels.KDSOrders{}
		// datetime(ko.completed) rather than a bare ko.completed desc: this
		// column has been written in more than one text format over time
		// (see recallOrderDatabase's completed-preservation attempt, which
		// silently falls back to an ISO "...T...Z" string on parse failure,
		// vs completeOrder's plain "YYYY-MM-DD HH:MM:SS" from getTimeSQL())
		// - SQLite's default TEXT ordering compares byte-for-byte, so a
		// mismatched separator character alone (' ' vs 'T') sorts every row
		// in one format before every row in the other regardless of actual
		// time, breaking "newest first" outright whenever the two are
		// mixed. datetime() normalizes both accepted formats to the same
		// canonical form before comparing, so the ORDER BY reflects real
		// chronological order no matter which format any given row was
		// written in.
		//
		// ", ko.id desc" tiebreaker: getTimeSQL() only has second precision,
		// so two orders bumped within the same second (routine during a
		// busy service, or any batch close) tie exactly on datetime(ko.
		// completed) - with no secondary key, SQLite fell back to whatever
		// order the rows happened to be scanned in, which is effectively
		// oldest-of-the-tied-group first, silently breaking "newest to
		// oldest" for that whole cluster. id desc breaks the tie by
		// creation order instead, which is what "newest" actually means
		// when completion time alone can't tell two orders apart.
		//
		// date('now','-30 MINUTE') was also wrong - date() truncates to a
		// bare YYYY-MM-DD with no time component, so the comparison really
		// only excluded rows completed before today, not before 30 minutes
		// ago (the header text's actual promise) - fixed to datetime(...,
		// 'localtime') so it's a genuine 30-minute window, matching
		// getTimeSQL()'s local-time (not UTC) timestamps.
		//
		// The strftime(...) = strftime(...) "completed today" check on
		// ko.created is also dropped: an order created just before midnight
		// but completed just after it (routine for a venue open past
		// midnight) has ko.created on the previous calendar day, so this
		// silently hid it from Recall Orders even though it was completed
		// moments ago - the 30-minute completed-recency filter above
		// already excludes anything actually stale, making this check both
		// redundant when it agrees and actively wrong when it doesn't.
		ENV.DB.Raw("select * from kds_orders as ko where ko.completed > datetime('now','-30 minutes','localtime') and ( (select count(1) from kds_items as ki where ki.kds_order_id = ko.id and ki.kds_screen_id = ? and ki.status = 'COMPLETE') > 0) order by datetime(ko.completed) desc, ko.id desc", kdsscreen.ID).Scan(&orders)
		for i := range orders {
			ENV.DB.Where("kds_order_id = ?", orders[i].ID).Find(&orders[i].Items)
		}
		return orders
	} else {
		return nil
	}
}

func getCompletedOrders() []kdsmodels.KDSOrders {
	orders := []kdsmodels.KDSOrders{}
	// See the matching comment in getCompletedOrdersScreen - datetime()
	// normalizes the two text formats this column can be written in so
	// mixed rows still sort by real chronological order.
	ENV.DB.Raw("select * from kds_orders as ko where strftime('%d-%m-%Y',ko.created) = strftime('%d-%m-%Y','now','localtime') and ko.completed > date('now','-30 MINUTE') and ( (select count(1) from kds_items as ki where ki.kds_order_id = ko.id and ki.status = 'COMPLETE') > 0) order by datetime(ko.completed) desc").Scan(&orders)
	for i := range orders {
		ENV.DB.Where("kds_order_id = ?", orders[i].ID).Find(&orders[i].Items)
	}
	return orders
}

func completeOrder(orderid int) {
	item := kdsmodels.KDSItems{}
	ENV.DB.Where("kds_order_id = ?", orderid).Where("status = ?", "OPEN").First(&item)
	if item.ID > 0 {
		//order is still active
	} else {
		order := kdsmodels.KDSOrders{}
		ENV.DB.Where("id = ?", orderid).First(&order)
		order.Status = "COMPLETE"
		abc := getTimeSQL()
		order.Completed = &abc
		order.Created = fixDateTime(order.Created)
		ENV.DB.Save(&order)
	}
}
func getDatabaseOrders(config kdsmodels.KDSConfig) []kdsmodels.KDSOrders {
	showDebug("GETTING ALL ORDERS")
	orders := []kdsmodels.KDSOrders{}
	// See todayOrDateUnparseable's own comment - confirmed live on this
	// exact query: corrupting one order's created to "" (to test the
	// frontend's oldest-first sort fix) made it silently drop out of every
	// terminal's board entirely, with no way to close it from the UI.
	ENV.DB.Where("status = ?", "OPEN").Where("strftime('%d-%m-%Y',created) = strftime('%d-%m-%Y', 'now', 'localtime')"+todayOrDateUnparseable).Find(&orders)
	//spew.Dump(orders)
	for i := range orders {
		ENV.DB.Where("kds_order_id = ?", orders[i].ID).Find(&orders[i].Items)
		for j := range orders[i].Items {
			ENV.DB.Where("kds_item_id = ?", orders[i].Items[j].ID).Where("status = ?", "OPEN").Find(&orders[i].Items[j].SubItems)
			ENV.DB.Where("id = ?", orders[i].Items[j].KDSScreenID).Find(&orders[i].Items[j].Screen)
		}
	}
	showDebug("END GETTING ALL ORDERS")
	return orders
}

/*
*
getTodaysDatabaseOrdersExport... Get Todays Database Orders
*/
func getTodaysDatabaseOrdersExport(config kdsmodels.KDSConfig) []kdsmodels.KDSExportOrders {
	showDebug("GETTING ALL ORDERS")
	orders := []kdsmodels.KDSExportOrders{}
	// See todayOrDateUnparseable's own comment - a malformed created value
	// should still show up in today's export, not silently vanish from it.
	ENV.DB.Where("strftime('%d-%m-%Y',created) = strftime('%d-%m-%Y', 'now', 'localtime')" + todayOrDateUnparseable).Find(&orders)
	for i := range orders {
		ENV.DB.Where("kds_order_id = ?", orders[i].ID).Find(&orders[i].Items)
		for j := range orders[i].Items {
			ENV.DB.Where("kds_item_id = ?", orders[i].Items[j].ID).Find(&orders[i].Items[j].SubItems)
			ENV.DB.Where("id = ?", orders[i].Items[j].KDSScreenID).Find(&orders[i].Items[j].Screen)
		}
	}
	showDebug("END GETTING ALL ORDERS")
	return orders
}

/*
*
getRangeDatabaseOrdersExport... Get A Range Of Database Orders
*/
func getRangeDatabaseOrdersExport(config kdsmodels.KDSConfig, start string, end string) []kdsmodels.KDSExportOrders {
	showDebug("GETTING ALL ORDERS")
	orders := []kdsmodels.KDSExportOrders{}
	ENV.DB.Where("created BETWEEN ? AND ?", start, end).Find(&orders)
	for i := range orders {
		ENV.DB.Where("kds_order_id = ?", orders[i].ID).Find(&orders[i].Items)
		for j := range orders[i].Items {
			ENV.DB.Where("kds_item_id = ?", orders[i].Items[j].ID).Find(&orders[i].Items[j].SubItems)
			ENV.DB.Where("id = ?", orders[i].Items[j].KDSScreenID).Find(&orders[i].Items[j].Screen)
		}
	}
	showDebug("END GETTING ALL ORDERS")
	return orders
}

/*
*
getAllDatabaseOrdersExport... Get All Database Orders
*/
func getAllDatabaseOrdersExport(config kdsmodels.KDSConfig) []kdsmodels.KDSExportOrders {
	showDebug("GETTING ALL ORDERS")
	orders := []kdsmodels.KDSExportOrders{}
	ENV.DB.Find(&orders)
	for i := range orders {
		ENV.DB.Where("kds_order_id = ?", orders[i].ID).Find(&orders[i].Items)
		for j := range orders[i].Items {
			ENV.DB.Where("kds_item_id = ?", orders[i].Items[j].ID).Find(&orders[i].Items[j].SubItems)
			ENV.DB.Where("id = ?", orders[i].Items[j].KDSScreenID).Find(&orders[i].Items[j].Screen)
		}
	}
	showDebug("END GETTING ALL ORDERS")
	return orders
}

/** GET ALL DATABASE ORDERS OPEN FOR THIS KDS */
func getAllDatabaseOrdersKDS() []kdsmodels.KDSOrders {
	showDebug("getAllDatabaseOrdersKDS")
	orders := []kdsmodels.KDSOrders{}
	// See todayOrDateUnparseable's own comment - this backs the full-board
	// fetch a terminal does on initial connect/reload, same failure mode
	// as getDatabaseOrders.
	ENV.DB.Where("status = ?", "OPEN").Where("strftime('%d-%m-%Y',created) = strftime('%d-%m-%Y', 'now', 'localtime')" + todayOrDateUnparseable).Find(&orders)
	for i := range orders {
		ENV.DB.Where("kds_order_id = ?", orders[i].ID).Find(&orders[i].Items)
		for j := range orders[i].Items {
			ENV.DB.Where("kds_item_id = ?", orders[i].Items[j].ID).Find(&orders[i].Items[j].SubItems)
			ENV.DB.Where("id = ?", orders[i].Items[j].KDSScreenID).Find(&orders[i].Items[j].Screen)
		}
	}
	return orders
}

/** GET SINGLE DATABASE ORDER OPEN FOR THIS KDS */
func getSingleDatabaseOrdersKDS(id int) kdsmodels.KDSOrders {
	showDebug("getAllDatabaseOrdersKDS")
	order := kdsmodels.KDSOrders{}
	ENV.DB.Where("id = ?", id).First(&order)
	if order.ID > 0 {
		ENV.DB.Where("kds_order_id = ?", order.ID).Find(&order.Items)
		for j := range order.Items {
			ENV.DB.Where("kds_item_id = ?", order.Items[j].ID).Find(&order.Items[j].SubItems)
			ENV.DB.Where("id = ?", order.Items[j].KDSScreenID).Find(&order.Items[j].Screen)
		}
	}
	return order
}

// todayOrDateUnparseableDATE is todayOrDateUnparseable's DATE()-flavored
// twin, for the call sites in this file built on DATE(x) = DATE(y) rather
// than strftime - same reasoning, same fix: DATE() also returns NULL for an
// unparseable value, and NULL never equals anything, so without this an
// order with a malformed created could be found by neither the peer-sync
// close/recall lookups below nor recallOrderDatabase/
// recallOrderScreenDatabase/recallOrderForPassScreen's own re-check further
// down - closable locally (those don't date-filter) but unrecallable
// afterward, and invisible to every other terminal's peer-sync copy,
// forever.
const todayOrDateUnparseableDATE = " OR DATE(created) IS NULL"

func getDatabaseOrderFromReference(reference string) kdsmodels.KDSOrders {
	showDebug("getDatabaseOrderFromReference")
	existing := kdsmodels.KDSOrders{}
	ENV.DB.Where("reference = ?", reference).Where("DATE(kds_orders.created) = DATE(CURRENT_TIMESTAMP, 'localtime')"+todayOrDateUnparseableDATE).Where("status =?", "OPEN").First(&existing)
	return existing
}

// getDatabaseOrderFromReferenceAnyStatus is getDatabaseOrderFromReference
// without the status=OPEN filter - used for applying a peer's recall to
// this terminal's own copy, where the whole point is finding an order
// that's currently COMPLETE (the OPEN-only lookup above would never match
// it) and reopening it.
//
// D08: without the OPEN filter to narrow things down, a reference reused
// later the same day (rare, but not impossible over a long service - see
// the ticket/Sally reference schemes' own doc comments, neither guarantees
// day-wide uniqueness) can match more than one row. Ordered by id desc and
// preferring whichever candidate's Table matches the caller's - recall is
// always about one specific physical table's ticket, and the table a
// recall actually originated from is known at the point it's first sent
// (see recallorderHandler), so it's the one piece of context available
// here that can actually distinguish "the order this recall really means"
// from "some other, unrelated order that happens to share a reference."
// table is deliberately advisory, not a hard filter: if it's blank (an
// older peer that hasn't sent it yet) or doesn't match any candidate, this
// still falls back to the most recent row rather than finding nothing -
// matching this function's original fail-open design (surfacing a
// possibly-wrong order is recoverable; silently finding nothing, given
// this is the only lookup capable of reopening a COMPLETE order at all,
// is not).
func getDatabaseOrderFromReferenceAnyStatus(reference string, table string) kdsmodels.KDSOrders {
	showDebug("getDatabaseOrderFromReferenceAnyStatus")
	candidates := []kdsmodels.KDSOrders{}
	ENV.DB.Where("reference = ?", reference).Where("DATE(kds_orders.created) = DATE(CURRENT_TIMESTAMP, 'localtime')"+todayOrDateUnparseableDATE).Order("id desc").Find(&candidates)
	if len(candidates) == 0 {
		return kdsmodels.KDSOrders{}
	}
	if table != "" {
		for _, candidate := range candidates {
			if strings.EqualFold(candidate.Table, table) {
				return candidate
			}
		}
	}
	return candidates[0]
}
func getDatabaseItemFromNameAndQTY(order kdsmodels.KDSOrders, itemname string, itemqty int, status string) kdsmodels.KDSItems {
	showDebug("getDatabaseItemFromName")
	existing := kdsmodels.KDSItems{}
	ENV.DB.Where("kds_order_id = ?", order.ID).Where("name = ?", itemname).Where("qty = ?", itemqty).Where("status = ?", status).First(&existing)
	return existing
}
func getDatabaseItemFromSuborder(order kdsmodels.KDSOrders, suborder int) kdsmodels.KDSItems {
	showDebug("getDatabaseItemFromName")
	existing := kdsmodels.KDSItems{}
	ENV.DB.Where("kds_order_id = ?", order.ID).Where("suborder = ?", suborder).Where("status = ?", "OPEN").First(&existing)
	return existing
}
func getDatabaseItemFromName(order kdsmodels.KDSOrders, item kdsmodels.KDSItems) kdsmodels.KDSItems {
	showDebug("getDatabaseItemFromName")
	existing := kdsmodels.KDSItems{}
	ENV.DB.Where("kds_order_id = ?", order.ID).Where("name = ?", item.Name).Where("qty = ?", item.Qty).Where("status = ?", "OPEN").First(&existing)
	return existing
}
func getDatabaseItemFromNameClosed(order kdsmodels.KDSOrders, item kdsmodels.KDSItems) kdsmodels.KDSItems {
	showDebug("getDatabaseItemFromName")
	existing := kdsmodels.KDSItems{}
	fmt.Println(order.ID)
	fmt.Println(item.Name)
	fmt.Println(item.Qty)
	ENV.DB.Where("kds_order_id = ?", order.ID).Where("name = ?", item.Name).Where("qty = ?", item.Qty).Where("status = ?", "COMPLETE").First(&existing)
	return existing
}

// itemOccurrenceInOrder returns this item's 1-based rank, by id (creation
// order), among items in the same order sharing the same Name+Qty - see the
// Occurrence field doc on kdsmodels.KDSItems for why this exists. Every
// terminal has its own auto-increment ids, so a remote status-change
// broadcast can only address an item by Name+Qty, which is ambiguous
// whenever an order has more than one matching item (e.g. two identical
// steaks on one ticket). Occurrence is stable across status changes since
// it's based purely on id ordering, not current status.
func itemOccurrenceInOrder(item kdsmodels.KDSItems) int {
	siblings := []kdsmodels.KDSItems{}
	ENV.DB.Where("kds_order_id = ?", item.KDSOrderID).Where("name = ?", item.Name).Where("qty = ?", item.Qty).Order("id ASC").Find(&siblings)
	for i := range siblings {
		if siblings[i].ID == item.ID {
			return i + 1
		}
	}
	return 1
}

// getDatabaseItemByOccurrence finds the item at the given 1-based Occurrence
// rank among items in this order sharing the same Name+Qty - the
// counterpart to itemOccurrenceInOrder, used on the receiving side to
// resolve which of several identical items a remote status change is for.
// Falls back to the first match if occurrence is out of range (e.g. sent by
// an older binary that doesn't populate Occurrence yet, or the two
// terminals' copies of the order have already diverged) - still better than
// dropping the update entirely.
func getDatabaseItemByOccurrence(orderID int, name string, qty int, occurrence int) kdsmodels.KDSItems {
	siblings := []kdsmodels.KDSItems{}
	ENV.DB.Where("kds_order_id = ?", orderID).Where("name = ?", name).Where("qty = ?", qty).Order("id ASC").Find(&siblings)
	if occurrence >= 1 && occurrence <= len(siblings) {
		return siblings[occurrence-1]
	}
	if len(siblings) > 0 {
		return siblings[0]
	}
	return kdsmodels.KDSItems{}
}

func openTheItemDB(itemid int) {
	item := kdsmodels.KDSItems{}
	order := kdsmodels.KDSOrders{}
	ENV.DB.Where("id = ?", itemid).First(&item)
	if item.ID > 0 {
		ENV.DB.Where("id = ?", item.KDSOrderID).First(&order)
		//Check if the order is still open , otherwise ignore
		if order.ID > 0 {
			if order.Status == "OPEN" {
				//ok we can open the item again
				item.Status = "OPEN"
				item.PrepSecs = int(0)
				item.PrepMins = int(0)
				ENV.DB.Save(&item)
				ENV.DB.Where("kds_item_id = ?", item.ID).Find(&item.SubItems)
				for j := range item.SubItems {
					item.SubItems[j].Status = "OPEN"
					ENV.DB.Save(&item.SubItems[j])
				}
			}
		}
	}
	notifyKDSOrders()
}
// closeTheItemDB marks a single item COMPLETE when a "DUPCLOSEITEM" event
// arrives via the screen1/screen2 NATS duplication path. It never
// auto-closes the order, even once every item is complete - same as
// closeItemsHandler and applyCloseRemoteItem (the peers/UDP equivalent of
// this function). Closing an order is always a deliberate Close Order
// action, never an automatic side effect of ticking off the last item -
// this used to auto-complete the order here (a leftover from before that
// rule existed), which meant marking every item done on a terminal reached
// via screen1/screen2 duplication would silently close the order on the
// terminal at the other end of that connection.
func closeTheItemDB(itemid int) {
	item := kdsmodels.KDSItems{}
	ENV.DB.Where("id = ?", itemid).First(&item)
	if item.ID > 0 {
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
		ENV.DB.Save(&item)
		ENV.DB.Where("kds_item_id = ?", item.ID).Find(&item.SubItems)
		for j := range item.SubItems {
			item.SubItems[j].Status = "COMPLETE"
			ENV.DB.Save(&item.SubItems[j])
		}
	}
	notifyKDSOrders()
}
// closeTheOrderDB is the legacy screen-scoped close (reachable only via the
// now-unused /closeorder/{id}/{screen} HTTP route - the live Electron/NATS
// flow uses closeOrderScreenDatabase/closeOrderDatabase instead, see
// natsCloseOrder). Kept working rather than deleted in case that route is
// ever revived. A screen name that doesn't match any real screen used to
// fall through to completing every item in the order regardless of screen -
// the exact bug this whole feature fixes elsewhere - so now it just closes
// nothing for that screen (closeOrderScreenDatabase's own no-match branch),
// matching the new terminals' behavior instead of silently closing
// everything on a typo/mismatch.
func closeTheOrderDB(orderid int, screenname string) kdsmodels.KDSOrders {
	return closeOrderScreenDatabase(orderid, screenname, true)
}

// closeOrderScreenDatabase completes the items on orderid that belong to
// screenname, plus any item stuck on a blank-named screen (an item whose
// screen marker failed to parse has no real screen name to ever be matched
// against, and would otherwise stay stuck OPEN forever - see
// LoadOrCreateScreen/getScreenSally's own comments on why that name is left
// blank rather than renamed), then - only when allowFinalize is true -
// re-evaluates the order's own status via completeOrder (which leaves it
// OPEN as long as any item anywhere - on this screen or another - is still
// OPEN). If screenname doesn't match any real screen, this completes
// nothing but still re-checks the order status when allowFinalize, since
// some other screen's own close may have been the one still pending.
//
// allowFinalize must be false for a REMOTE-received close (see
// applyCloseOrderScreenRemote) - see that call site's own comment for why:
// a genuinely different terminal's items being complete already is not
// this terminal's business to act on.
func closeOrderScreenDatabase(orderid int, screenname string, allowFinalize bool) kdsmodels.KDSOrders {
	orders := kdsmodels.KDSOrders{}
	ENV.DB.Where("id = ?", orderid).First(&orders)
	if orders.ID == 0 {
		fmt.Println("ORDER NOT FOUND")
		return orders
	}
	relevant := false
	screens := []kdsmodels.KDSScreens{}
	ENV.DB.Where("LOWER(name) = LOWER(?) OR name = ?", screenname, "").Find(&screens)
	if len(screens) > 0 {
		screenIDs := make([]int, len(screens))
		for i := range screens {
			screenIDs[i] = screens[i].ID
		}
		if orderHasItemsMatching(orderid, "kds_screen_id IN ?", screenIDs) {
			relevant = true
		}
		completeOrderItems(orderid, "kds_screen_id IN ?", screenIDs)
	}
	// See closeOrderForPassScreen's comment - an item with no screen set at
	// all belongs to no terminal, so whoever presses Close Order first
	// (pass or not) sweeps it, same as the blank-named-screen sweep above
	// already does for the case that DOES have a screen row to match.
	orphanScope := "kds_screen_id IS NULL OR kds_screen_id = 0 OR kds_screen_id NOT IN (SELECT id FROM kds_screens)"
	if orderHasItemsMatching(orderid, orphanScope) {
		relevant = true
	}
	completeOrderItems(orderid, orphanScope)
	// Only re-evaluate/finalize THIS terminal's own copy of the order if
	// this close action actually concerned something on it (a matching
	// screen item, or an orphan) AND it's safe to finalize at all here
	// (allowFinalize - see this function's own doc comment). See
	// orderHasItemsMatching's doc comment for the live cross-terminal
	// incident (headchef closing its own screen was finalizing/closing a
	// completely unrelated terminal's order row - e.g. bar's - purely
	// because THAT row's own item had already been marked complete
	// earlier, independent of this action).
	if relevant && allowFinalize {
		completeOrder(orders.ID)
	}
	ENV.DB.Where("id = ?", orderid).First(&orders)
	return orders
}

// closeOrderForPassScreen force-closes, on orderid, only the items whose
// screen is passScreenName itself, is assigned to it
// (KDSScreens.AssignedPassTerminal, set via that section screen's own
// Client Config "Pass Terminal" dropdown), or is a blank-named orphan (same
// sweep closeOrderScreenDatabase already does). Item status is irrelevant
// here - that's what makes this a "force" close, letting a pass terminal
// bump its whole section without every sub-terminal having individually
// marked its own items complete first - but unlike the previous
// closeOrderDatabase-for-everything behavior, it now stops at this
// terminal's own section instead of sweeping the entire order, so a
// completely separate terminal (e.g. a Standalone Mode Bar, on the same
// order as Kitchen's food) is never touched by a Kitchen pass screen's
// bump. An earlier version of this same idea (closeOrderForPassTerminal,
// since removed) was dropped because an unassigned section became
// permanently unclosable by the pass terminal's bump - that risk doesn't
// reapply here: AssignedPassTerminal blank means the section still bumps
// its own orders itself (see its own doc comment), so a not-yet-assigned
// sub-terminal simply falls back to closing its own items locally rather
// than being silently swept by, or silently blocked from, a pass screen it
// was never actually assigned to.
func closeOrderForPassScreen(orderid int, passScreenName string, allowFinalize bool) kdsmodels.KDSOrders {
	orders := kdsmodels.KDSOrders{}
	ENV.DB.Where("id = ?", orderid).First(&orders)
	if orders.ID == 0 {
		fmt.Println("ORDER NOT FOUND")
		return orders
	}
	relevant := false
	screens := []kdsmodels.KDSScreens{}
	ENV.DB.Where("LOWER(name) = LOWER(?) OR LOWER(assigned_pass_terminal) = LOWER(?) OR name = ?", passScreenName, passScreenName, "").Find(&screens)
	if len(screens) > 0 {
		screenIDs := make([]int, len(screens))
		for i := range screens {
			screenIDs[i] = screens[i].ID
		}
		if orderHasItemsMatching(orderid, "kds_screen_id IN ?", screenIDs) {
			relevant = true
		}
		completeOrderItems(orderid, "kds_screen_id IN ?", screenIDs)
	}
	// Per explicit instruction: an item with no screen set at all
	// (kds_screen_id 0/NULL, or pointing at a screen row that's since been
	// deleted) always gets force-closed too, regardless of section - it
	// doesn't belong to any section, so nothing above would ever match it,
	// and completeOrder below refuses to finish the order while ANY item
	// anywhere is still OPEN. Without this, an order with one of these
	// stayed permanently stuck: visible in pass mode (which doesn't filter
	// by status), gone from Standalone Mode's own board (its own items
	// already done), but Close Order silently doing nothing either way.
	orphanScope := "kds_screen_id IS NULL OR kds_screen_id = 0 OR kds_screen_id NOT IN (SELECT id FROM kds_screens)"
	if orderHasItemsMatching(orderid, orphanScope) {
		relevant = true
	}
	completeOrderItems(orderid, orphanScope)
	// Only re-evaluate/finalize THIS terminal's own copy of the order if
	// this close action actually concerned something on it.
	//
	// Live incident this fixes: a tab shares one `reference` across
	// terminals, but each terminal keeps its OWN separate order row with
	// only ITS OWN items on it (confirmed live: headchef's row for a round
	// had food items only, bar's row for the very same round had only its
	// own drink item - never a shared row). completeOrder's own check is
	// (correctly) unscoped *within* a row, since one terminal can
	// legitimately have one row spanning several of its own screens - but
	// that means ANY close broadcast reaching a peer via
	// applyCloseOrderScreenRemote (matched purely by reference, since
	// every terminal has its own auto-increment ids) would finalize that
	// peer's ENTIRE row the instant nothing on it was left OPEN, even when
	// this specific close action never touched anything on that row at
	// all. Gating on relevant (a matching screen item, or an orphan) means
	// a close scoped to a screen this row has nothing to do with is now a
	// true no-op for it, matching this function's own doc comment above
	// ("a completely separate terminal... is never touched").
	//
	// allowFinalize adds a second, stronger guard on top of relevant: even
	// when this row DOES have a matching item (so relevant is true), a
	// REMOTE-received close (see applyCloseOrderScreenRemote) must never
	// decide "the whole order is done" on this terminal's behalf just
	// because every item on this row happens to already be complete - only
	// THIS terminal's own local Close Order press may finalize its own
	// copy of an order - a remote peer's close only ever syncs the
	// matching item's own status, never the order-level completion.
	if relevant && allowFinalize {
		completeOrder(orders.ID)
	}
	ENV.DB.Where("id = ?", orderid).First(&orders)
	return orders
}

// orderHasItemsMatching reports whether orderid has any item matching the
// given scope condition - used to gate completeOrder's finalization check in
// closeOrderScreenDatabase/closeOrderForPassScreen so a screen-scoped close
// with nothing to do with this terminal's own copy of the order (no matching
// screen item, no orphan) can't finalize/close it purely as a side effect of
// items that were already complete before this specific close action ran.
func orderHasItemsMatching(orderid int, scopeQuery string, scopeArgs ...interface{}) bool {
	var count int64
	query := ENV.DB.Model(&kdsmodels.KDSItems{}).Where("kds_order_id = ?", orderid)
	query = query.Where(scopeQuery, scopeArgs...)
	query.Count(&count)
	return count > 0
}

// completeOrderItems marks COMPLETE every item on orderid additionally
// matching the given scope condition (a kds_screen_id filter, in practice) -
// used by closeOrderScreenDatabase to scope a non-pass terminal's bump to
// its own screen only.
func completeOrderItems(orderid int, scopeQuery string, scopeArgs ...interface{}) {
	mainitems := []kdsmodels.KDSItems{}
	query := ENV.DB.Where("kds_order_id = ?", orderid)
	query = query.Where(scopeQuery, scopeArgs...)
	query.Find(&mainitems)
	for i := range mainitems {
		mainitems[i].Status = "COMPLETE"
		prims := dif2times(fixDateTime(mainitems[i].Created))
		mainitems[i].PrepSecs = int(math.Round(prims[1]))
		mainitems[i].PrepMins = int(math.Round(prims[0]))
		mainitems[i].Created = fixDateTime(mainitems[i].Created)
		ENV.DB.Save(&mainitems[i])
		ENV.DB.Where("kds_item_id = ?", mainitems[i].ID).Find(&mainitems[i].SubItems)
		for j := range mainitems[i].SubItems {
			mainitems[i].SubItems[j].Status = "COMPLETE"
			ENV.DB.Save(&mainitems[i].SubItems[j])
		}
	}
}

// screenUpsertMu serializes upsertScreenPassAssignment's check-then-write
// against kds_screens (no unique index on name) - without it, two
// near-simultaneous calls for the same screenname (e.g. two announce/relay
// requests landing on separate goroutines within the same second, as
// happens across an 8-terminal cold start) can both SELECT zero rows before
// either INSERT commits, leaving duplicate rows for the same screen name.
var screenUpsertMu sync.Mutex

// upsertScreenPassAssignment records which pass terminal (possibly none)
// bumps screenname's orders, creating the screen row locally if this
// terminal has never seen that screen name before (e.g. a pass terminal
// applying an assignment for a section it doesn't itself prepare items for).
func upsertScreenPassAssignment(screenname string, passterminal string) {
	if screenname == "" {
		return
	}
	screenUpsertMu.Lock()
	defer screenUpsertMu.Unlock()
	screen := kdsmodels.KDSScreens{}
	ENV.DB.Where("LOWER(name) = LOWER(?)", screenname).First(&screen)
	if screen.ID == 0 {
		screen.Name = screenname
	}
	screen.AssignedPassTerminal = passterminal
	ENV.DB.Save(&screen)
}

// passTerminalUpsertMu is screenUpsertMu's counterpart for kds_pass_terminals
// - same select-then-save shape, same lack of a unique index on name.
var passTerminalUpsertMu sync.Mutex

// upsertPassTerminalAnnounce records that a pass-mode terminal named `name`
// is present on-site, for section terminals' own "Pass Terminal" config
// dropdown (getPassTerminalsHandler) - see announceAsPassTerminal.
func upsertPassTerminalAnnounce(name string) {
	if name == "" {
		return
	}
	passTerminalUpsertMu.Lock()
	defer passTerminalUpsertMu.Unlock()
	terminal := kdsmodels.KDSPassTerminals{}
	ENV.DB.Where("LOWER(name) = LOWER(?)", name).First(&terminal)
	if terminal.ID == 0 {
		terminal.Name = name
	}
	terminal.LastSeen = getTimeSQL()
	ENV.DB.Save(&terminal)
}
func recallOrderDatabase(orderid int) kdsmodels.KDSOrders {
	order := kdsmodels.KDSOrders{}
	ENV.DB.Where("id = ?", orderid).Where("DATE(created) = DATE(CURRENT_TIMESTAMP, 'localtime')"+todayOrDateUnparseableDATE).First(&order)
	if order.ID > 0 {
		items := []kdsmodels.KDSItems{}
		ENV.DB.Where("kds_order_id = ?", orderid).Where("status = ?", "COMPLETE").Find(&items)
		for i := range items {
			items[i].Status = "OPEN"
			ENV.DB.Save(&items[i])
		}
		// Created/Completed are left untouched - both are already in
		// getTimeSQL()'s plain "YYYY-MM-DD HH:MM:SS" form from when they
		// were first set (order creation / completeOrder), never the ISO
		// "...T...Z" form fixDateTime/fixDateTime2 expect as input, so
		// re-running either here always fails to parse and silently
		// replaces the real original timestamp with the current time -
		// corrupting the order's actual elapsed-time/completion history on
		// every recall for no benefit (recalling only needs to flip Status
		// back to OPEN; Completed becomes meaningful again next time this
		// order actually completes, via completeOrder).
		order.Status = "OPEN"
		order.PassStatus = "OPEN"
		ENV.DB.Save(&order)
	}
	return order
}

// recallOrderScreenDatabase is recallOrderDatabase's screen-scoped
// counterpart, mirroring closeOrderScreenDatabase vs closeOrderDatabase -
// only reopens screenname's own completed items (falling back to every
// completed item if screenname doesn't match any real screen, same
// leave-it-be default as the close side), rather than the whole order
// regardless of screen. Used for a non-pass terminal's own Recall Orders
// press, which should only ever undo its own screen's items - reopening
// another screen's items it has no business touching would be the same
// class of bug closeOrderScreenDatabase exists to avoid on the close side.
func recallOrderScreenDatabase(orderid int, screenname string) kdsmodels.KDSOrders {
	order := kdsmodels.KDSOrders{}
	ENV.DB.Where("id = ?", orderid).Where("DATE(created) = DATE(CURRENT_TIMESTAMP, 'localtime')"+todayOrDateUnparseableDATE).First(&order)
	if order.ID > 0 {
		screen := kdsmodels.KDSScreens{}
		ENV.DB.Where("LOWER(name) = LOWER(?)", screenname).Find(&screen)
		items := []kdsmodels.KDSItems{}
		if screen.ID > 0 {
			ENV.DB.Where("kds_order_id = ?", orderid).Where("status = ?", "COMPLETE").Where("kds_screen_id = ?", screen.ID).Find(&items)
			// See closeOrderScreenDatabase's comment - an item with no screen
			// set at all belongs to no terminal, so it's swept here too rather
			// than only ever being reachable by the "unrecognized screen name"
			// fallback below.
			noScreenItems := []kdsmodels.KDSItems{}
			ENV.DB.Where("kds_order_id = ?", orderid).Where("status = ?", "COMPLETE").Where("kds_screen_id IS NULL OR kds_screen_id = 0 OR kds_screen_id NOT IN (SELECT id FROM kds_screens)").Find(&noScreenItems)
			items = append(items, noScreenItems...)
		} else {
			ENV.DB.Where("kds_order_id = ?", orderid).Where("status = ?", "COMPLETE").Find(&items)
		}
		for i := range items {
			items[i].Status = "OPEN"
			ENV.DB.Save(&items[i])
		}
		// See recallOrderDatabase's comment - Created/Completed deliberately
		// left untouched.
		order.Status = "OPEN"
		order.PassStatus = "OPEN"
		ENV.DB.Save(&order)
	}
	return order
}

// recallOrderForPassScreen is recallOrderDatabase's pass-terminal
// counterpart, mirroring closeOrderForPassScreen vs closeOrderDatabase - it
// used to reopen every completed item on the order with no screen filter at
// all, meaning a Kitchen pass screen's Recall Order press would also
// silently reopen an already-completed Bar drink on the same order. This
// scopes the reopen the same way the close side is now scoped: passScreenName
// itself, any screen assigned to it (KDSScreens.AssignedPassTerminal), or a
// blank-named orphan.
func recallOrderForPassScreen(orderid int, passScreenName string) kdsmodels.KDSOrders {
	order := kdsmodels.KDSOrders{}
	ENV.DB.Where("id = ?", orderid).Where("DATE(created) = DATE(CURRENT_TIMESTAMP, 'localtime')"+todayOrDateUnparseableDATE).First(&order)
	if order.ID > 0 {
		screens := []kdsmodels.KDSScreens{}
		ENV.DB.Where("LOWER(name) = LOWER(?) OR LOWER(assigned_pass_terminal) = LOWER(?) OR name = ?", passScreenName, passScreenName, "").Find(&screens)
		items := []kdsmodels.KDSItems{}
		if len(screens) > 0 {
			screenIDs := make([]int, len(screens))
			for i := range screens {
				screenIDs[i] = screens[i].ID
			}
			ENV.DB.Where("kds_order_id = ?", orderid).Where("status = ?", "COMPLETE").Where("kds_screen_id IN ?", screenIDs).Find(&items)
		}
		// See closeOrderForPassScreen's comment - a completed item with no
		// screen set at all is always swept too, so a pass-mode recall can
		// undo everything its matching force-close touched.
		noScreenItems := []kdsmodels.KDSItems{}
		ENV.DB.Where("kds_order_id = ?", orderid).Where("status = ?", "COMPLETE").Where("kds_screen_id IS NULL OR kds_screen_id = 0 OR kds_screen_id NOT IN (SELECT id FROM kds_screens)").Find(&noScreenItems)
		items = append(items, noScreenItems...)
		for i := range items {
			items[i].Status = "OPEN"
			ENV.DB.Save(&items[i])
		}
		// See recallOrderDatabase's comment - Created/Completed deliberately
		// left untouched.
		order.Status = "OPEN"
		order.PassStatus = "OPEN"
		ENV.DB.Save(&order)
	}
	return order
}
func closeOrderDatabase(orderid int) kdsmodels.KDSOrders {
	orders := kdsmodels.KDSOrders{}
	mainitems := []kdsmodels.KDSItems{}
	ENV.DB.Where("id = ?", orderid).First(&orders)
	if orders.ID > 0 {
		ENV.DB.Where("kds_order_id = ?", orderid).Find(&mainitems)
		for i := range mainitems {
			mainitems[i].Status = "COMPLETE"
			prims := dif2times(fixDateTime(mainitems[i].Created))
			mainitems[i].PrepSecs = int(math.Round(prims[1]))
			mainitems[i].PrepMins = int(math.Round(prims[0]))
			mainitems[i].Created = fixDateTime(mainitems[i].Created)
			ENV.DB.Save(&mainitems[i])
			ENV.DB.Where("kds_item_id = ?", mainitems[i].ID).Find(&mainitems[i].SubItems)
			for j := range mainitems[i].SubItems {
				mainitems[i].SubItems[j].Status = "COMPLETE"
				ENV.DB.Save(&mainitems[i].SubItems[j])
			}
		}
		completeOrder(orders.ID)
	} else {
		fmt.Println("ORDER NOT FOUND")
	}
	return orders
}
