package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/reverent911/kds-golang-lib/pkg/generic/helper"
	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
)

func openItemsRemoteHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("openItemsRemoteHandler")
	item := kdsmodels.KDSItems{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&item)
	if err != nil {
		fmt.Println(err)
	}
	applyOpenRemoteItem(item)
	w.Write([]byte("OK"))
	notifyKDSOrders()
}

// applyOpenRemoteItem re-opens an item on this terminal's own copy of an
// order, matched by order Reference + item Name + Qty + Occurrence (the
// item's rank among same-Name+Qty siblings) since every terminal has its own
// auto-increment ids and an order can have more than one identical item.
// Shared by the HTTP remote endpoint (single "Remote"/peers targets) and the
// UDP discovery listener (no IP needed).
func applyOpenRemoteItem(item kdsmodels.KDSItems) {
	order := getDatabaseOrderFromReference(item.Reference)
	//spew.Dump(item)
	//fmt.Println("FOUND ORDER : " + helper.IntToString(order.ID))
	if order.ID > 0 {
		if order.Status == "OPEN" {
			//fmt.Println("ORDER STATUS : " + order.Status)
			theitem := getDatabaseItemByOccurrence(order.ID, item.Name, item.Qty, item.Occurrence)
			if theitem.ID > 0 {
				//fmt.Println("FOUND ITEM : " + helper.IntToString(theitem.ID))
				theitem.Status = "OPEN"
				theitem.PrepSecs = int(0)
				theitem.PrepMins = int(0)
				// Reopening never resumes production by itself - matches the
				// frontend, which asks the user before starting production
				// again rather than doing it silently.
				theitem.InProduction = false
				theitem.ProductionStartedAt = nil
				theitem.ProductionEndedAt = nil
				// Reopening an item pulls it back off the Order Display
				// Screen too - it's no longer actually ready.
				theitem.PushedToOds = false
				theitem.PushedToOdsAt = nil
				//fmt.Println("SAVING ITEM")
				ENV.DB.Save(&theitem)
				ENV.DB.Where("kds_item_id = ?", theitem.ID).Find(&theitem.SubItems)
				for j := range theitem.SubItems {
					theitem.SubItems[j].Status = "OPEN"
					ENV.DB.Save(&theitem.SubItems[j])
				}
			} else {
				fmt.Println("NOT FOUND ITEM")
			}
		}
	}
}

func closeRemoteItemHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("closeRemoteItemHandler")
	item := kdsmodels.KDSItems{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&item)
	if err != nil {
		fmt.Println(err)
	}
	applyCloseRemoteItem(item)
	w.Write([]byte("OK"))
	notifyKDSOrders()
}

// applyCloseRemoteItem marks an item COMPLETE on this terminal's own copy of
// an order, matched by order Reference + item Name + Qty + Occurrence (see
// applyOpenRemoteItem). Shared by the HTTP remote endpoint and the UDP
// discovery listener.
func applyCloseRemoteItem(remoteItem kdsmodels.KDSItems) {
	orderhere := getDatabaseOrderFromReference(remoteItem.Reference)
	//fmt.Println(item)
	if orderhere.ID > 0 {
		item := getDatabaseItemByOccurrence(orderhere.ID, remoteItem.Name, remoteItem.Qty, remoteItem.Occurrence)
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
			// Completing an item announces it on the Order Display Screen
			// automatically - same rule as closeItemsHandler, applied here so
			// a terminal that receives this completion (rather than being the
			// one that made it) ends up with the same pushed state.
			if !item.PushedToOds {
				now := getTimeSQL()
				item.PushedToOds = true
				item.PushedToOdsAt = &now
			}
			ENV.DB.Save(&item)
			ENV.DB.Where("kds_item_id = ?", item.ID).Find(&item.SubItems)
			for j := range item.SubItems {
				item.SubItems[j].Status = "COMPLETE"
				ENV.DB.Save(&item.SubItems[j])
			}
			// Same as closeItemsHandler - never auto-close on this server's
			// own copy of the order, even when the completion arrived from a
			// duplicated remote screen. Closing is always a deliberate
			// Close Order action now.
		}
	}
}

// closeItemsHandler marks a single item complete (the "Only the Head Chef
// pass KDS may mark an item Complete" requirement) - the frontend already
// refuses to make this call on a non-pass terminal (KDS.close_item's own
// passmode check), but that's only a UI courtesy; without the matching
// check here, anything able to reach this terminal's own local HTTP API
// directly could still mark an item complete on a section terminal. This
// never gates the remote-sync path (closeRemoteItemHandler/
// applyCloseRemoteItem, a different endpoint) - a peer's already-validated
// close must still apply to this terminal's own mirrored copy regardless of
// this terminal's own passmode setting.
func closeItemsHandler(w http.ResponseWriter, r *http.Request) {
	if !ENV.Config.Flags.Passmode {
		http.Error(w, "only the pass terminal may mark an item complete", http.StatusForbidden)
		return
	}
	vars := mux.Vars(r)
	itemid := vars["id"]
	item := kdsmodels.KDSItems{}
	order := kdsmodels.KDSOrders{}
	ENV.DB.Where("id = ?", itemid).First(&item)
	if item.ID > 0 {
		item.Status = "COMPLETE"
		prims := dif2times(fixDateTime(item.Created))
		item.PrepSecs = int(math.Round(prims[1]))
		item.PrepMins = int(math.Round(prims[0]))
		item.Created = fixDateTime(item.Created)
		if item.InProduction {
			// Ticking an item as produced always ends its production, even
			// though the frontend never makes a separate clearproduction call -
			// otherwise the stale in_production flag keeps the orange
			// "in production" styling/timer alive on top of the green COMPLETE
			// tag, on this terminal and on every synced copy.
			now := getTimeSQL()
			item.InProduction = false
			item.ProductionEndedAt = &now
		}
		// Completing an item announces it on the Order Display Screen
		// immediately - no separate "push" step. See kdsitems.go PushedToOds.
		now := getTimeSQL()
		item.PushedToOds = true
		item.PushedToOdsAt = &now
		ENV.DB.Save(&item)
		ENV.DB.Where("kds_item_id = ?", item.ID).Find(&item.SubItems)
		for j := range item.SubItems {
			item.SubItems[j].Status = "COMPLETE"
			ENV.DB.Save(&item.SubItems[j])
		}
		ENV.DB.Where("id = ?", item.KDSOrderID).First(&order)
		// Ticking off the last item never auto-closes the order anymore, in
		// either mode - it stays open, showing each item's completed tag,
		// until the user closes it explicitly via the Close Order button
		// (whose visibility the frontend decides: always in pass mode, only
		// once every item is complete otherwise).
		item.Reference = order.Reference
		item.Occurrence = itemOccurrenceInOrder(item)
		itemjs, _ := json.Marshal(&item)
		sendIfNeeded("DUPCLOSEITEM", itemjs)
		sendIfCallBackPOST("closeitemremote", itemjs)
		sendToPeers("closeitemremote", itemjs)
		broadcastUDP("closeitemremote", itemjs)
		//sendIfCallBack("closeitemremote", url.QueryEscape(order.Reference)+"/"+helper.IntToString(item.Qty)+"/"+url.QueryEscape(item.Name))
	}
	w.Write([]byte("OK"))
	notifyKDSOrders()

}

func openItemsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	itemid := vars["id"]
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
				// Reopening never resumes production by itself - the frontend
				// asks the user before starting production again.
				item.InProduction = false
				item.ProductionStartedAt = nil
				item.ProductionEndedAt = nil
				// Reopening an item pulls it back off the Order Display
				// Screen too - it's no longer actually ready.
				item.PushedToOds = false
				item.PushedToOdsAt = nil
				ENV.DB.Save(&item)
				ENV.DB.Where("kds_item_id = ?", item.ID).Find(&item.SubItems)
				for j := range item.SubItems {
					item.SubItems[j].Status = "OPEN"
					ENV.DB.Save(&item.SubItems[j])
				}
			}
		}
		item.Reference = order.Reference
		item.Occurrence = itemOccurrenceInOrder(item)
		itemjs, _ := json.Marshal(&item)
		sendIfNeeded("DUPOPENITEM", itemjs)
		sendIfCallBackPOST("openitemremote", itemjs)
		sendToPeers("openitemremote", itemjs)
		broadcastUDP("openitemremote", itemjs)
		//sendIfCallBack("openitemremote", order.Reference+"/"+helper.IntToString(item.SubOrder))
	}
	w.Write([]byte("OK"))
	notifyKDSOrders()
}

func closeOrdersHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderid := vars["id"]
	screenname := vars["screen"]
	orders := closeTheOrderDB(helper.StringToInt(orderid), screenname)
	w.Write([]byte("OK"))
	notifyKDSOrders()
	notifyPrepTime(screenname)
	js, _ := json.Marshal(&orders)
	notifyEvent("CLOSEDORDER", string(js))
	sendIfNeeded("DUPCLOSEORDER", js)
	sendIfCallBack("closeremote", orders.Reference)
	sendToPeers("closeorderremote", js)
	broadcastUDP("closeorderremote", js)
	/*if len(ENV.Config.Duplicate.Remote) > 1 {
		fmt.Println("Closing Remote Order")
		_, err := http.Get("http://" + ENV.Config.Duplicate.Remote + ":2020/closeremote/" + orders.Reference)
		if err != nil {
			fmt.Println(err)
		}
	}*/
}

// closeOrderRemoteHandler receives a "close this order" fan-out from another
// KDS terminal - typically the pass-mode terminal, via sendToPeers - so every
// other KDS on site closes the same order. Unlike closeRemoteOrdersHandler
// (addressed by a local order id, used by the legacy single-"Remote" GET
// callback), this is a POST body addressed by the order's Reference, since
// each terminal has its own auto-increment ids for the same order.
func closeOrderRemoteHandler(w http.ResponseWriter, r *http.Request) {
	order := kdsmodels.KDSOrders{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&order)
	if err != nil {
		fmt.Println(err)
	}
	applyCloseOrderRemote(order)
	w.Write([]byte("OK"))
}

// applyCloseOrderRemote closes the given order on this terminal's own copy,
// matched by Reference - if this terminal has never seen that order, it's a
// no-op. Shared by the HTTP remote endpoint and the UDP discovery listener.
//
// Uses closeOrderDatabase (unconditional - every item in the order), NOT
// closeTheOrderDB(id, screenname) - that variant only completes items whose
// screen name matches the string passed in, and ENV.Config.Client.Terminal
// (this terminal's own screen/terminal name) almost always names one of
// THIS terminal's own real screens. Passing it here would silently filter
// out every item that isn't on that one screen, so an order with items
// spread across screens (the normal case) would never actually finish
// closing on the receiving terminal even though this handler reports
// success. A remote/duplicate close means "the whole order is done," the
// same thing closeOrderDatabase already does for a local Close Order click.
func applyCloseOrderRemote(order kdsmodels.KDSOrders) {
	orderhere := getDatabaseOrderFromReference(order.Reference)
	if orderhere.ID > 0 {
		closeOrderDatabase(orderhere.ID)
		notifyKDSOrders()
		notifyPrepTime(ENV.Config.Client.Terminal)
	} else {
		fmt.Println("Invalid Order")
	}
}

// closeOrderScreenRemoteHandler receives a screen-scoped close-order fan-out
// from another terminal on-site (see natsCloseOrder's
// sendToPeers("closeorderscreenremote", ...)) and applies the same scoped
// completion to this terminal's own copy of the order.
func closeOrderScreenRemoteHandler(w http.ResponseWriter, r *http.Request) {
	closemsg := kdsmodels.KDSOrderScreenClose{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&closemsg)
	if err != nil {
		fmt.Println(err)
	}
	applyCloseOrderScreenRemote(closemsg)
	w.Write([]byte("OK"))
}

// isDeferredToPassTerminal reports whether THIS terminal's own local
// kds_screens has at least one screen whose AssignedPassTerminal names
// passTerminalName - i.e. whether this terminal genuinely relies on that
// pass terminal to decide when its own orders are done. This is the exact
// existing relationship a sub-screen's own Client Config "Pass Terminal"
// dropdown records (see KDSScreens.AssignedPassTerminal's own doc comment)
// - reused here as the one authoritative signal for whether a remote
// close may finalize this terminal's own copy of an order.
func isDeferredToPassTerminal(passTerminalName string) bool {
	var count int64
	ENV.DB.Model(&kdsmodels.KDSScreens{}).Where("LOWER(assigned_pass_terminal) = LOWER(?)", passTerminalName).Count(&count)
	return count > 0
}

// applyCloseOrderScreenRemote applies the given close on this terminal's own
// copy, matched by Reference - if this terminal has never seen that order,
// it's a no-op. Replays the exact same scoped completion the originating
// terminal applied - closeOrderForPassScreen (scoped to closemsg.Screen's
// own section) when the close came from a pass terminal's bump, same as
// natsCloseOrder; closeOrderScreenDatabase (this screen's own items only)
// otherwise - so every terminal's copy of the order ends up with the same
// items completed, no more and no less.
//
// allowFinalize is only true when THIS terminal is actually, genuinely
// assigned to defer closing to closemsg.Screen (isDeferredToPassTerminal) -
// e.g. pizza/burgerandcold/bakery/pnp are each assigned to headchef, so
// headchef's own bump legitimately finalizes their orders too (that's the
// whole point of pass-terminal assignment - a sub-screen has no other way
// to close its own orders). Without a genuine assignment relationship, a
// remote close only ever syncs the matching item's own status and never
// finalizes - confirmed live: bar force-closing its own screen (already-
// complete drink item) reached kitchen's completely independent copy of
// the same shared-reference row (which happened to carry a duplicate of
// that same item) and finalized/closed kitchen's ticket too, purely
// because kitchen's own food item had ALSO already been tapped complete -
// even though kitchen itself never pressed Close Order, and bar was never
// assigned as kitchen's pass terminal in the first place.
func applyCloseOrderScreenRemote(closemsg kdsmodels.KDSOrderScreenClose) {
	orderhere := getDatabaseOrderFromReference(closemsg.Reference)
	if orderhere.ID > 0 {
		allowFinalize := isDeferredToPassTerminal(closemsg.Screen)
		if closemsg.Passmode {
			closeOrderForPassScreen(orderhere.ID, closemsg.Screen, allowFinalize)
		} else {
			closeOrderScreenDatabase(orderhere.ID, closemsg.Screen, allowFinalize)
		}
		notifyKDSOrders()
		notifyPrepTime(ENV.Config.Client.Terminal)
	} else {
		fmt.Println("Invalid Order")
	}
}

// screenPassAssignmentRemoteHandler receives another terminal's own
// screen-to-pass-terminal assignment (see announceOwnScreenPassAssignment)
// and records it in this terminal's own local copy of kds_screens, so the
// assignment (still shown in Client Config's pass terminal listing) is
// consistent across every terminal's own database, regardless of which
// terminal's UI the assignment was originally made on. No longer gates
// closing - see natsCloseOrder.
func screenPassAssignmentRemoteHandler(w http.ResponseWriter, r *http.Request) {
	assignment := kdsmodels.KDSScreenPassAssignment{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&assignment)
	if err != nil {
		fmt.Println(err)
		return
	}
	upsertScreenPassAssignment(assignment.Screen, assignment.Passterminal)
}

// passTerminalAnnounceRemoteHandler receives another pass-mode terminal's
// self-announcement (see announceAsPassTerminal) so this terminal can list
// it as an option in its own "Pass Terminal" config dropdown.
func passTerminalAnnounceRemoteHandler(w http.ResponseWriter, r *http.Request) {
	announce := kdsmodels.KDSPassTerminalAnnounce{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&announce)
	if err != nil {
		fmt.Println(err)
		return
	}
	upsertPassTerminalAnnounce(announce.Name)
}

// passTerminalStaleAfter is how long a pass terminal can go without
// re-announcing itself before it's dropped from the "Pass Terminal"
// dropdown as no longer on-site.
const passTerminalStaleAfter = 6 * time.Minute

// listAvailablePassTerminals returns every pass terminal this terminal has
// heard announce itself recently enough to still trust - shared by
// getPassTerminalsHandler (passive, whatever's already accumulated) and
// rediscoverPassTerminalsHandler (after actively asking everyone to
// re-announce right now). A LastSeen this terminal can't parse is kept
// rather than hidden, so a formatting mismatch can't silently erase a real
// terminal from the list.
func listAvailablePassTerminals() []kdsmodels.KDSPassTerminals {
	all := []kdsmodels.KDSPassTerminals{}
	ENV.DB.Find(&all)
	cutoff := time.Now().Add(-passTerminalStaleAfter)
	fresh := []kdsmodels.KDSPassTerminals{}
	for _, t := range all {
		seen, err := time.ParseInLocation("2006-01-02 15:04:05", t.LastSeen, time.Local)
		if err != nil || seen.After(cutoff) {
			fresh = append(fresh, t)
		}
	}
	return fresh
}

// getPassTerminalsHandler lists every pass terminal this terminal has heard
// announce itself, for the "Pass Terminal" dropdown on a section's own
// config page (GET http://127.0.0.1:2020/getpassterminals).
func getPassTerminalsHandler(w http.ResponseWriter, r *http.Request) {
	terminals := listAvailablePassTerminals()
	js, _ := json.Marshal(&terminals)
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

// rediscoverPassTerminalsHandler backs the Client Config screen's
// "Rediscover Pass Terminals" button (GET
// http://127.0.0.1:2020/rediscoverpassterminals) - triggers an immediate
// discovery round over every delivery path (requestPassTerminalRediscovery),
// gives LAN replies a moment to actually arrive and be recorded
// (upsertPassTerminalAnnounce), then returns the resulting list in the same
// shape as getPassTerminalsHandler so the dropdown can repopulate from a
// single round trip instead of the caller guessing when to re-fetch.
func rediscoverPassTerminalsHandler(w http.ResponseWriter, r *http.Request) {
	requestPassTerminalRediscovery()
	time.Sleep(1500 * time.Millisecond)
	terminals := listAvailablePassTerminals()
	js, _ := json.Marshal(&terminals)
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

// discoverPingRemoteHandler receives a "please re-announce yourself now"
// request from another terminal's own Rediscover Pass Terminals button (the
// HTTP-peers leg of requestPassTerminalRediscovery) and responds by
// immediately re-running this terminal's own announce cycle instead of
// waiting for its next periodic 2-minute tick. No payload to decode -
// receiving this at all is the whole signal.
func discoverPingRemoteHandler(w http.ResponseWriter, r *http.Request) {
	asyncAnnouncePassTerminalInfo()
	w.Write([]byte("OK"))
}

func closeRemoteOrdersHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderid := vars["id"]
	if len(orderid) > 0 {
		orderhere := getDatabaseOrderFromReference(orderid)
		if orderhere.ID > 0 {
			// Same reasoning as applyCloseOrderRemote above - unconditional,
			// not screen-filtered by this terminal's own name.
			orders := closeOrderDatabase(orderhere.ID)
			notifyKDSOrders()
			notifyPrepTime(ENV.Config.Client.Terminal)
			js, _ := json.Marshal(&orders)
			notifyEvent("CLOSEDORDER", string(js))
			natsReload()
		} else {
			fmt.Println("Invalid Order")
		}
	} else {
		fmt.Println("Invalid Message")
	}
}
