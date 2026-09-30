package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/reverent911/kds-golang-lib/pkg/generic/helper"
	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
)

// stripControlChars drops non-printable control bytes (e.g. the stray
// leading 0x1D "group separator" the printer format puts in front of some
// lines, including its own "vb" end-of-ticket marker) so text comparisons
// against known literals aren't broken by them.
func stripControlChars(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}

func ProcessMessage(obj kdsmodels.KDSPollObject, content []byte, root string, dblocation string) {
	fmt.Println(content)
	var db = createSQLITEDatabase(obj.Config, nil, DBLocation)
	defer closeConnection(db)
	obj.Neworder.Status = "OPEN"
	obj.Neworder.PassStatus = "OPEN"
	obj.Neworder.Created = helper.UtilgetTimeSQL()
	obj.Itemsection = false
	obj.Cancontinue = true
	obj.Foundprofile = false
	obj.Foundstaff = false
	obj.Foundreference = false
	obj.Foundscreen = false
	obj.Foundtable = false
	obj.Voiditem = false
	obj.Foundpagernr = false
	obj.Nextlineisorder = false
	obj.Nextlineiscollectioncode = false
	obj.Itemadded = false
	obj.Foundconsumer = false
	obj.Founddelivery = false
	obj.Foundcell = false
	foundTransfer := false
	// A "TRANSFERRED TO <table>" line is immediately followed on the ticket
	// by the name of the item that's moving - but only for a line-item
	// transfer, and critically that item name line has no "N x " quantity
	// prefix (unlike every normal item line), so it never matches the item
	// regexes below and obj.Items stays empty either way. This is the only
	// signal that distinguishes "this specific item moved" from "the whole
	// order moved" (no item name line at all). Every real example seen so
	// far transfers exactly one item per ticket, so this captures only the
	// single line immediately following the transfer line - deliberately
	// not "keep reading until some terminator", since the ticket's own "vb"
	// end marker is unreliable to match exactly (stray control-byte
	// prefixes) and any miss there would swallow the rest of the ticket's
	// raw byte padding as a bogus extra "item".
	awaitingTransferItemName := false
	var transferItemName string
	var sourceTableFromTicket string
	r := bytes.NewReader(content)
	scanner := bufio.NewScanner(r)
	extrateller := 8
	foundOrder := false
	for scanner.Scan() {
		text := strings.ToLower(scanner.Text())

		// Courses
		course := GetMatchAndReturn(text, courseReg)
		if course != "" {
			if obj.Currentitem.Qty > 0 {
				obj.Items = append(obj.Items, obj.Currentitem)
				obj.Currentitem = kdsmodels.KDSMainItems{}
			}
			itemc := kdsmodels.KDSMainItems{}
			itemc.Qty = 1
			itemc.Name = "COURSE : " + course
			ShowDebug("COURSE : " + course)
			//fmt.Println(obj.Items)
			obj.Items = append(obj.Items, itemc)
			ShowDebug("COURSE ADDED")
		}

		if !foundTransfer {
			transfer, transfertable := IsThisTransfer(text)
			if transfer {
				// obj.Neworder.Table currently holds whatever the ticket's
				// own "Table N" line set earlier - that's the table this
				// transfer is FROM, about to be overwritten below with the
				// destination. Keep it: if this ticket's reference doesn't
				// match any known order (the source table's tile may have
				// been created under a different reference - e.g. by an
				// earlier line-item transfer's own synthetic "-XFERn" one,
				// which the POS never sees or reuses), this is the fallback
				// for finding it.
				sourceTableFromTicket = obj.Neworder.Table
				obj.Neworder.Table = transfertable
				foundTransfer = transfer
				awaitingTransferItemName = true
			}
		} else if awaitingTransferItemName {
			awaitingTransferItemName = false
			cleaned := strings.TrimSpace(stripControlChars(text))
			if cleaned != "" && cleaned != "vb" {
				transferItemName = strings.TrimSpace(stripControlChars(scanner.Text()))
			}
		}

		if obj.Nextlineisorder {
			ShowDebug("NEXTINLINE")
			table := strings.ReplaceAll(text, ")", "")
			table = strings.ReplaceAll(table, "(", "")
			table = strings.TrimSpace(table)
			// Some tickets prefix this line with an extra cosmetic token
			// (e.g. "E Table 1", where "E" isn't part of the table name) -
			// if the line contains "table"/"order"/"tafel"/"tisch", use
			// that match instead of the raw line so such prefixes are
			// dropped. Falls back to the raw line for tickets where the
			// table is just a bare number with none of those words.
			if matched := GetMatchAndReturn(table, tableReg); matched != "" {
				table = matched
			}
			obj.Neworder.Table = table
			obj.Nextlineisorder = false
			// Without this, the generic table-matching block further down
			// still runs on this same line: tableReg only matches from
			// "table" onward, so on a line like "E Table 1" it extracts
			// just "table 1" - different from what's already set here
			// ("e table 1"), so its "only append if different" check
			// treats that as a second, distinct table and appends it,
			// producing "e table 1table 1" (shown as "E TABLE 1TABLE 1").
			// The table was already correctly captured above; there's
			// nothing left for that block to find on this line.
			obj.Foundtable = true
		}

		if !obj.Founddelivery {
			delivery := GetMatchAndReturn(text, deliverTime)
			if delivery != "" {
				ShowDebug("Delivery: " + delivery)
				obj.Founddelivery = true
				obj.Neworder.Delivery = delivery
			}
		}

		if !obj.Foundprofile {
			profile := GetMatchAndReturn(text, profileReg)
			if profile != "" {
				ShowDebug("PROFILE: " + profile)
				obj.Foundprofile = true
				obj.Neworder.Profile = profile
				obj.Nextlineisorder = true
			}
		}
		if !obj.Foundscreen {
			screen := GetMatchAndReturn(text, screenReg)
			if screen != "" {
				ShowDebug("SCREEN: " + screen)
				obj.Foundscreen = true
				obj.Screenkds = LoadOrCreateScreen(db, screen)
			}
		}

		if !obj.Foundstaff {
			staff := GetMatchAndReturn(text, staffReg)
			if staff != "" {
				ShowDebug("STAFF: " + staff)
				obj.Foundstaff = true
				obj.Neworder.Waiter = staff
			}
		}

		if !obj.Foundreference {
			reference := GetMatchAndReturn(text, referenceReg)
			ShowDebug(reference)
			if reference != "" {
				ShowDebug("REFERENCE: " + reference)
				obj.Foundreference = true
				obj.Neworder.Reference = reference
			}
		}

		if !obj.Foundcell {
			cell := GetMatchAndReturn(text, customerCell)
			if cell != "" {
				ShowDebug("CELL : " + cell)
				obj.Foundcell = true
				obj.Neworder.PassStatus = cell
			} else {
				cell = GetMatchAndReturn(text, customerCellUnsafe)
				if cell != "" {
					ShowDebug("CELL : " + cell)
					obj.Foundcell = true
					obj.Neworder.PassStatus = cell
				}
			}
		}

		if !obj.Foundtable {
			table := GetMatchAndReturn(text, tableReg)
			if table != "" {
				ShowDebug("TABLE: " + table)
				obj.Foundtable = true
				if obj.Neworder.Table != table {
					obj.Neworder.Table += table
				}
			} else {
				table2 := GetMatchAndReturn(text, tableModified)
				if table2 != "" {
					ShowDebug("TABLE MODIFIED: " + table2)
					obj.Foundtable = true
					if obj.Neworder.Table != table2 {
						obj.Neworder.Table += table2
					}
				}
			}

		}
		if !foundOrder && !obj.Foundtable {
			table := GetMatchAndReturn(text, orderOnlyReg)
			if table != "" {
				ShowDebug("ORDER: " + table)
				foundOrder = true
				obj.Neworder.Table = table + " <br />"
				//spew.Dump("ORDER: " + obj.Neworder.Table)
			}
		}
		if !obj.Foundpagernr {
			pagernr := GetMatchAndReturn(text, pagernrReg)
			if pagernr != "" {
				pageN1 := GetInts(pagernr)
				ShowDebug("pagernr: " + pageN1)
				obj.Foundpagernr = true
				obj.Neworder.Table = "RNR: " + pageN1
			}
		}

		// @TODO : QTY OF VOIDS
		if obj.Voiditem {
			getVoidItemsMatching(text, &obj)
			obj.Voiditem = false
		} else {
			obj.Voiditem = getIsThisVoid(text)
			if !obj.Voiditem {
				if !GetItemsMatching(text, &obj) {
					if obj.Itemsection {
						GetSubItemMatches(text, proditemReg, &obj)                    //Sequences 1 x ABC
						GetProductionInstructions(text, productionInstructions, &obj) //Production Instructions 1 ABC
						endOfItem(text, endofmainReg, &obj)
					}
				}
			}
		}
		if obj.Nextlineiscollectioncode {
			obj.Neworder.Waiter = obj.Neworder.Waiter + "\n(C-CODE:" + GetInts(text) + ")"
			obj.Nextlineiscollectioncode = false
		}
		if getIsThisCollectionCode(text) {
			obj.Nextlineiscollectioncode = true
		}
		if obj.Foundconsumer {
			//Consumer was found everything from here should be included.
			ShowDebug("extra consumer: " + text)
			if extrateller > 0 {
				//spew.Dump(text)
				text = strings.Replace(text, "\x00", "", -1)
				text = strings.Replace(text, "\x1d", "", -1)
				text = strings.Replace(text, "a ", "", -1)
				text = strings.Replace(text, "\\", "", -1)
				text = strings.Replace(text, "\"", "", -1)
				text = strings.TrimSpace(text)
				if strings.Contains(text, "+") || strings.Contains(text, "contacts") {
					text = repNums(text)
				}
				if strings.Contains(text, "customer") || strings.Contains(text, "consumer") {
					text = ""
				}
				if text != "" && text != "a" && text != "a " && len(text) > 2 {
					//fmt.Println("ADDING: " + text)
					obj.Neworder.Consumer += ("<br />" + text)
				}
				extrateller -= 1
			}
		}
		if !obj.Foundconsumer {
			consumer := GetMatchAndReturn(text, consumerName)
			if consumer != "" {
				ShowDebug("Consumer: " + consumer)
				obj.Foundconsumer = true
				consumer = "######"
				obj.Neworder.Consumer = consumer
			}
		}

	} // End Of Scanning

	if obj.Currentitem.Qty > 0 {
		obj.Items = append(obj.Items, obj.Currentitem)
		obj.Currentitem = kdsmodels.KDSMainItems{}
	}
	var suborderforitems = 0
	//fmt.Println("LOOKING FOR")
	//fmt.Println(obj.Neworder.Reference)
	existing := CheckExistingOrder(db, obj.Neworder.Reference)
	if existing.ID == 0 && foundTransfer && sourceTableFromTicket != "" {
		// The ticket's own reference doesn't match any order we know about.
		// This happens when the tile actually sitting at the source table
		// was itself created by an earlier line-item transfer, under a
		// synthetic "-XFERn" reference the POS has never seen and so can't
		// reuse - from the POS's perspective, "table 200" may now carry a
		// different (or no) reference entirely. Fall back to whichever open
		// order is actually sitting at the table this transfer claims to be
		// FROM, so a transfer chained off an earlier split still finds it
		// instead of silently creating an orphaned, itemless order.
		existing = FindOpenOrderAtTable(db, sourceTableFromTicket)
	}
	usedTableFallbackForVoid := false
	if existing.ID == 0 && len(obj.Voiditems) > 0 && len(obj.Items) == 0 && obj.Neworder.Table != "" {
		// A void-only ticket (no new items of its own) can carry a reference
		// the POS minted just for that void transaction, unrelated to the
		// reference of the order actually holding the item - seen in
		// practice: voiding an item on a table that had earlier received a
		// line-item transfer produced a "VOIDED ORDER" ticket under a brand
		// new reference, the only link back to the real order being the
		// "Table N" line both share. Without this, the void silently spins
		// up an empty orphan order instead of ever reaching the real item.
		existing = FindOpenOrderAtTable(db, obj.Neworder.Table)
		usedTableFallbackForVoid = true
	}

	// Only a genuinely brand new order/tile (set below, at db.Create(&obj.Neworder)
	// in the existing.ID == 0 branch) needs telling other terminals about -
	// see broadcastNewOrder. An update to an order this terminal already
	// has doesn't need it: every OTHER terminal either already knows about
	// that order too (it got the original broadcast), or never will, and
	// re-broadcasting the same order on every ticket update would just be
	// noise.
	wasNewOrder := false

	if existing.ID > 0 {
		if foundTransfer && transferItemName != "" {
			// Line-item transfer: this ticket named a destination table AND
			// the specific item that's moving, not the whole order.
			// Renaming existing.Table here (the old behavior) would drag
			// every OTHER item still on this order along with it, since
			// they all share the same order row/tile. Instead, spin up a
			// distinct new order/tile at the destination table for just the
			// transferred item, and leave the original order/tile (and all
			// its other items) exactly where they are.
			sourceTable := existing.Table
			// Reuse whatever's already open at the destination table -
			// regardless of how it got there - instead of only recognizing
			// a tile created by an earlier transfer from this exact same
			// source order. That narrower match (by a "-XFER" suffix on
			// this source order's own reference) used to miss two real
			// cases: (1) the destination table already had its own,
			// organic open order (never involved in any transfer) sitting
			// there when this transfer arrived, and (2) a multi-item
			// transfer where the SOURCE order itself was a different
			// order than THIS transfer's source but shared the same
			// destination table. Both produced a redundant second tile at
			// a table that already had one open. FindOpenOrderAtTable
			// already exists for exactly this "any open order at this
			// table" lookup (used elsewhere for void fallback) and is
			// already scoped to status='OPEN' - a bumped/completed order
			// at the destination correctly won't match, so a new tile
			// still gets created in that case, same as before.
			transferOrder := FindOpenOrderAtTable(db, obj.Neworder.Table)
			if transferOrder.ID != 0 && transferOrder.TransferNote == "" {
				// Reused an order that wasn't itself created by a transfer
				// (or was, but from a different source) - it has no banner
				// yet to tell staff an item just arrived from elsewhere.
				// Only set this if it's not already showing one: an order
				// that's already a transfer child keeps announcing its
				// OWN original source rather than being overwritten by
				// whichever transfer happens to reuse it next.
				transferOrder.TransferNote = "transferred from " + sourceTable
				db.Save(&transferOrder)
			}
			if transferOrder.ID == 0 {
				transferOrder = kdsmodels.KDSOrders{}
				transferOrder.Status = "OPEN"
				transferOrder.PassStatus = "OPEN"
				transferOrder.Table = obj.Neworder.Table
				transferOrder.Waiter = existing.Waiter
				// Keep the normal profile heading (e.g. "sit down") - this
				// ticket has its own "* ... *" profile line same as any
				// other, already captured into obj.Neworder.Profile above.
				// The transfer itself is called out separately via
				// TransferNote, so the frontend can render it as its own
				// distinct banner above the items instead of replacing the
				// heading.
				transferOrder.Profile = obj.Neworder.Profile
				transferOrder.TransferNote = "transferred from " + sourceTable
				transferOrder.Created = helper.UtilgetTimeSQL()
				ordercounter := NewOrderForToday(db)
				ordercounter++
				transferOrder.OrderFortoday = ordercounter
				// Every KDS terminal runs its own poller against the same POS
				// print feed and writes to its own local DB independently -
				// there's no cross-machine coordination at this layer. A
				// counter-based suffix (NewOrderForToday, ordercounter above)
				// is LOCAL to this machine's own order history, so two
				// terminals processing the identical real-world transfer
				// would each compute a different suffix and end up with
				// different references for what's supposed to be the same
				// tile. That breaks every later cross-machine sync for this
				// tile (production/complete/void), since the webserver's
				// peer/UDP sync matches purely by reference. The suffix must
				// instead be derived only from data every terminal received
				// identically from the same ticket: the original reference
				// plus the destination table name.
				sanitizedTable := strings.ToUpper(strings.Join(strings.Fields(obj.Neworder.Table), ""))
				transferOrder.Reference = fmt.Sprintf("%s-XFER-%s", existing.Reference, sanitizedTable)
				db.Create(&transferOrder)
			}
			obj.Neworder = transferOrder
			// The transferred item name never went through the normal item
			// regexes (no "N x " quantity prefix on this ticket, see the
			// capture logic above), so it was never added to obj.Items -
			// create its row directly here instead of relying on the
			// "PROCESS ITEMS" loop below.
			var newMain kdsmodels.KDSItems
			newMain.Qty = 1
			newMain.QtyRaw = 1
			// Lowercase, matching every other item creation path in this
			// file (they all extract from `text`, which is lowercased
			// before any parsing) - transferItemName instead preserves the
			// ticket's original case, and other matching against this item
			// (e.g. a later void) compares names case-sensitively.
			newMain.Name = strings.ToLower(RemoveNonUTF(transferItemName))
			newMain.Status = "OPEN"
			newMain.KDSScreenID = obj.Screenkds.ID
			newMain.KDSOrderID = transferOrder.ID
			newMain.Created = helper.UtilgetTimeSQL()
			db.Create(&newMain)
			// The item just moved to its own tile above - without this it
			// would still show as OPEN on the original order/tile too,
			// looking like it was duplicated rather than moved. Status is
			// TRANSFERRED rather than VOID - it was never cancelled, so the
			// frontend omits it entirely instead of showing a VOID badge
			// (see the filter at the top of HTML_GENERATE_ORDER_ITEMS).
			// Items are stored lowercase (see GetItemsMatching, which
			// matches against the already-lowercased scan line), so match
			// case-insensitively against the transfer ticket's
			// original-case item name.
			var sourceItem kdsmodels.KDSItems
			db.Where("kds_order_id = ?", existing.ID).
				Where("name = ?", strings.ToLower(RemoveNonUTF(transferItemName))).
				Where("status = ?", "OPEN").
				Order("id DESC").
				First(&sourceItem)
			if sourceItem.ID > 0 {
				sourceItem.Status = "TRANSFERRED"
				db.Save(&sourceItem)
			}
		} else {
			if foundTransfer {
				//whole table transfer (no items on this ticket) - every item
				//already on the order is moving together, so just relabel
				//the existing order/tile in place. Same tile, not a new one,
				//so TransferNote needs setting explicitly here too - the
				//line-item path only sets it once at the NEW child tile's
				//creation, which never happens for a whole-table move.
				//Without this the frontend's transfer banner (gated on
				//order.transfer_note being non-empty) never appears even
				//though the ticket clearly said "transferred to".
				sourceTable := existing.Table
				existing.Table = obj.Neworder.Table
				existing.TransferNote = "transferred from " + sourceTable
			}
			obj.Neworder = existing
			db.Save(&obj.Neworder)
			obj.Neworder.SubOrders++
			suborderforitems = GetSubOrderForItems(db, obj.Neworder.ID, obj.Screenkds.ID)
			suborderforitems++
		}
	} else {
		ordercounter := NewOrderForToday(db)
		ordercounter++
		obj.Neworder.OrderFortoday = ordercounter
		ordercounter++
		db.Create(&obj.Neworder)
		wasNewOrder = true
	}

	//process voids
	// A void ticket carries no transfer info of its own - it just names the
	// item and the order's original reference, same as it always has. If
	// that item was since moved to its own tile by a line-item transfer,
	// looking only at obj.Neworder.ID (the original order) misses it
	// entirely, since the original's own copy is marked TRANSFERRED (and
	// hidden), not OPEN - the void ticket has no way to reference the
	// synthetic child tile it doesn't know exists. Search the whole family
	// (original + any open transfer children) so the void reaches wherever
	// the item actually still lives.
	var voidCandidateIDs []int
	if usedTableFallbackForVoid {
		// The reference-based family (existing.Reference/obj.Neworder.ID)
		// only covers whichever single order FindOpenOrderAtTable happened
		// to pick above - if a second, unrelated order is also open at that
		// same table right now (e.g. two different tables both transferred
		// onto this one), narrowing to just the picked order's family could
		// silently miss the item or, worse, match the wrong one of two
		// same-named items. Search every open order's family at this table
		// instead, so the match is decided by which one actually has the
		// named item.
		voidCandidateIDs = OpenOrderFamilyIDsAtTable(db, obj.Neworder.Table)
	} else {
		voidFamilyReference := obj.Neworder.Reference
		if existing.ID > 0 {
			voidFamilyReference = existing.Reference
		}
		voidCandidateIDs = OpenOrderFamilyIDs(db, voidFamilyReference, obj.Neworder.ID)
	}
	for g := range obj.Voiditems {
		if len(obj.Voiditems[g].Name) > 0 {
			// status = OPEN matters once the family can span more than one
			// order: without it, whichever candidate has the lowest id wins
			// regardless of state, which could match the original order's
			// now-TRANSFERRED (hidden) copy instead of the live OPEN one
			// actually sitting on the transfer-child tile.
			var itemvoiditem kdsmodels.KDSItems
			db.Where("kds_order_id IN (?)", voidCandidateIDs).Where("name = ?", obj.Voiditems[g].Name).Where("qty = ?", obj.Voiditems[g].Qty).Where("status = ?", "OPEN").First(&itemvoiditem)
			if itemvoiditem.ID > 0 {
				ShowDebug("FOUND EXACT MATCH")
				itemvoiditem.Status = "VOID"
				itemvoiditem.Created = helper.UtilgetTimeSQL()
				db.Save(&itemvoiditem)
			} else {
				db.Where("kds_order_id IN (?)", voidCandidateIDs).Where("name = ?", obj.Voiditems[g].Name).Where("status = ?", "OPEN").First(&itemvoiditem)
				if itemvoiditem.ID > 0 {
					ShowDebug("FOUND A MATCH")
					// Wherever the matched item actually lives (possibly a
					// transfer child, not obj.Neworder) is also where its
					// split-off VOID remainder row belongs.
					targetOrderID := itemvoiditem.KDSOrderID
					itemvoiditem.Qty = itemvoiditem.Qty - obj.Voiditems[g].Qty
					if itemvoiditem.Qty < 1 {
						itemvoiditem.Qty = 1
					}
					itemvoiditem.Created = helper.UtilgetTimeSQL()
					db.Save(&itemvoiditem)

					var newMain kdsmodels.KDSItems
					newMain.Qty = obj.Voiditems[g].Qty
					newMain.Name = obj.Voiditems[g].Name
					newMain.Status = "VOID"
					newMain.KDSScreenID = obj.Screenkds.ID
					newMain.KDSOrderID = targetOrderID
					newMain.Created = helper.UtilgetTimeSQL()
					newMain.PrepMins = 0
					newMain.PrepSecs = 0
					newMain.SubOrder = suborderforitems
					db.Create(&newMain)
				}
			}
		}
	} // end voids

	for j := range obj.Items { // PROCESS ITEMS
		if len(obj.Items[j].Name) > 0 {
			var newMain kdsmodels.KDSItems
			newMain.Qty = obj.Items[j].Qty
			newMain.QtyRaw = obj.Items[j].QtyRaw
			newMain.Name = RemoveNonUTF(obj.Items[j].Name)
			if strings.Contains(newMain.Name, "COURSE :") {
				newMain.Status = "COURSE"
			} else {
				newMain.Status = "OPEN"
			}
			newMain.KDSScreenID = obj.Screenkds.ID
			newMain.KDSOrderID = obj.Neworder.ID
			newMain.Created = helper.UtilgetTimeSQL()
			if strings.Contains(newMain.Name, "kiddies") {
				newMain.IsKiddies = 1
			} else if strings.Contains(newMain.Name, "kids") {
				newMain.IsKiddies = 1
			} else {
				newMain.IsKiddies = 0
			}
			newMain.PrepMins = 0
			newMain.PrepSecs = 0
			newMain.SubOrder = suborderforitems
			db.Create(&newMain)

			if newMain.ID > 0 {
				for k := range obj.Items[j].Subs {
					var newSubMain kdsmodels.KDSSubitems
					newSubMain.Qty = obj.Items[j].Subs[k].Qty
					newSubMain.QtyRaw = obj.Items[j].Subs[k].QtyRaw
					newSubMain.Name = obj.Items[j].Subs[k].Name
					newSubMain.Status = "OPEN"
					newSubMain.KDSItemID = newMain.ID
					db.Create(&newSubMain)
				}
			}
		}
	} // END ITEMS
	if wasNewOrder {
		// Tell every other terminal on the LAN this order/tile now exists,
		// with its items - see udp_broadcast.go. Every terminal runs its own
		// poller against the same ticket feed and would otherwise number
		// this order independently on its own local daily counter; this is
		// what keeps the "Chk N" number consistent across every terminal
		// that shows this ticket, including a terminal (an Order Display
		// Screen) that never ingests tickets itself at all.
		broadcastNewOrder(db, obj.Neworder.ID)
	}
	obj.Itemadded = true
	if obj.Itemadded {
		var kdsmessage kdsmodels.KDSTerminalMessage
		kdsmessage.HWID = obj.Config.Client.Serial
		b, _ := json.Marshal(&obj.Neworder)
		//myerror.HandelGlobalErrorAPP(err, myerror.GetSentryServices().KDSPOLLER, myerror.GetNewRelicServices().KDSPOLLER, NewRelicApp)
		kdsmessage.Message = string(b)
		kdsmessage.Type = "ORDERCREATED"
		b, _ = json.Marshal(&kdsmessage)
		//myerror.HandelGlobalErrorAPP(err, myerror.GetSentryServices().KDSPOLLER, myerror.GetNewRelicServices().KDSPOLLER, NewRelicApp)
		//NCglobal.Publish("system.messages", b)
		//NC.Publish("neworder", b)
		helper.KDSNATSPublishGlobal(NCglobal, "system.messages", b)
		helper.KDSNATSPublish(NC, "neworder", b)
		// Was `go NotifyServer(...)` - fire-and-forget, not awaited before
		// ProcessMessage returns. NotifyServer is what actually tells this
		// terminal's own webserver to push a fresh "updates" snapshot to
		// the board; the board only ever learns an order exists through
		// that snapshot, and only ever shows items with status OPEN. That
		// left a real race: if literally anything completed this order
		// (e.g. a multi-screen ticket whose other screen's terminal
		// bumped its own portion fast, finishing the whole order) before
		// the goroutine actually got scheduled and its HTTP call
		// actually landed, the board's very first snapshot of this order
		// already showed it complete - it would never render as an OPEN
		// tile at all, only ever showing up later in Recall Orders,
		// looking exactly like it had been bumped before anyone saw it.
		// Awaiting it here still returns well within NotifyServer's own
		// 10s timeout for a same-machine HTTP call, and closes that
		// window down to the single HTTP round-trip instead of leaving
		// it open-ended on goroutine scheduling.
		NotifyServer(obj.Config)
	}
}
