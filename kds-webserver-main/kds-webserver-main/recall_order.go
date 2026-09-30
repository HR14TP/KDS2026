package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/getsentry/sentry-go"
	"github.com/gorilla/mux"
	"github.com/reverent911/kds-golang-lib/pkg/generic/helper"
	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
)

func GetcompletedHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	orderid := vars["id"]
	orders := getCompletedOrdersScreen(orderid)
	js, err := json.Marshal(orders)
	if err != nil {
		sentry.CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

// recallorderHandler is the live Recall Orders press (see
// KDS.message_recallorder). A pass-mode terminal's recall reopens its own
// section (recallOrderForPassScreen: its own screen, plus every screen
// assigned to it) - the same section-scoped "whole ticket" semantics as its
// Close Order force-close (natsCloseOrder/closeOrderForPassScreen), since a
// multi-screen order it recalls was originally force-closed as a unit too,
// but it never reaches into a completely separate terminal like Bar. A
// non-pass terminal stays scoped to its own screen
// (recallOrderScreenDatabase), matching closeOrderScreenDatabase.
//
// Either way, the recall is then fanned out to every other terminal on-site
// (sendToPeers/broadcastUDP, same mechanism natsCloseOrder already uses) so
// each one reopens its OWN copy of the order too - without this, only the
// terminal where Recall Orders was actually pressed ever saw the order
// reopen; every other terminal the order originally showed on (any screen
// that had completed items on it) kept its own copy marked COMPLETE
// forever, with no way to reopen it from there.
func recallorderHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderid := vars["id"]
	screenname := vars["screen"]
	passmode := vars["passmode"] == "true"

	var order kdsmodels.KDSOrders
	if passmode {
		order = recallOrderForPassScreen(helper.StringToInt(orderid), screenname)
	} else {
		order = recallOrderScreenDatabase(helper.StringToInt(orderid), screenname)
	}

	w.Write([]byte("OK"))
	notifyKDSOrders()

	if order.ID > 0 {
		recallmsg := kdsmodels.KDSOrderScreenRecall{Reference: order.Reference, Screen: screenname, Passmode: passmode}
		json1, _ := json.Marshal(&recallmsg)
		sendToPeers("recallorderscreenremote", json1)
		broadcastUDP("recallorderscreenremote", json1)
	}
}

// recallOrderScreenRemoteHandler receives a recall fan-out from another
// terminal on-site (see recallorderHandler's sendToPeers) and applies the
// same recall to this terminal's own copy of the order.
func recallOrderScreenRemoteHandler(w http.ResponseWriter, r *http.Request) {
	recallmsg := kdsmodels.KDSOrderScreenRecall{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&recallmsg)
	if err != nil {
		fmt.Println(err)
	}
	applyRecallOrderScreenRemote(recallmsg)
	w.Write([]byte("OK"))
}

// applyRecallOrderScreenRemote reopens the given order on this terminal's
// own copy, matched by Reference (getDatabaseOrderFromReferenceAnyStatus,
// not the OPEN-only lookup used elsewhere - the whole point here is finding
// an order that's currently COMPLETE) - if this terminal has never seen
// that order, it's a no-op. Replays the exact same logic the originating
// terminal applied, same as applyCloseOrderScreenRemote does for closes.
func applyRecallOrderScreenRemote(recallmsg kdsmodels.KDSOrderScreenRecall) {
	orderhere := getDatabaseOrderFromReferenceAnyStatus(recallmsg.Reference)
	if orderhere.ID > 0 {
		if recallmsg.Passmode {
			recallOrderForPassScreen(orderhere.ID, recallmsg.Screen)
		} else {
			recallOrderScreenDatabase(orderhere.ID, recallmsg.Screen)
		}
		notifyKDSOrders()
	} else {
		fmt.Println("Invalid Order")
	}
}
