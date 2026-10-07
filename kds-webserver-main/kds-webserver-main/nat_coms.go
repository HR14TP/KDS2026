package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/nats-io/nats.go"
	"github.com/reverent911/kds-golang-lib/pkg/generic/helper"
	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsactions"
	"github.com/reverent911/kds-golang-lib/pkg/kds/kdshelper"
	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
)

var NC *nats.Conn
var NCglobal *nats.Conn
var conerror error
var NC1 *nats.Conn
var NC2 *nats.Conn

type RangeData struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

func sendIfNeeded(subject string, data []byte) {
	if NC1 != nil && NC1.IsConnected() {
		fmt.Println("Screen1 Duplication Send")
		helper.KDSNATSPublish(NC1, subject, data)
	}
	if NC2 != nil && NC2.IsConnected() {
		fmt.Println("Screen2 Duplication Send")
		helper.KDSNATSPublish(NC2, subject, data)
	}
}

// peerHostPort appends the default peer port (2020, this service's own
// hardcoded listen port for every normal production deployment) only when
// host doesn't already carry one - existing configs just store a bare host/
// IP and keep working unchanged, while a host explicitly written as
// "host:port" (e.g. two terminals sharing one machine on different ports,
// only really relevant for local testing) is respected as-is instead of
// having ":2020" blindly appended on top of it.
func peerHostPort(host string) string {
	if strings.Contains(host, ":") {
		return host
	}
	return host + ":2020"
}

func sendIfCallBack(subject string, data string) {
	if len(ENV.Config.Duplicate.Remote) > 1 {
		fmt.Println("REMOTE : " + subject)
		fmt.Println("REMOTE : " + "http://" + peerHostPort(ENV.Config.Duplicate.Remote) + "/" + subject + "/" + data)
		_, err := http.Get("http://" + peerHostPort(ENV.Config.Duplicate.Remote) + "/" + subject + "/" + data)
		if err != nil {
			fmt.Println(err)
		}
	}
}
// sendToPeers fans a change out to every other KDS terminal at the site,
// not just the 2 NATS mirror screens + 1 legacy remote that "Duplication"
// supports. It folds the legacy single Remote target into the same list so
// existing configs keep working. Each peer is notified concurrently and
// failures are logged and ignored, same fire-and-forget style as the
// existing duplication callbacks.
func sendToPeers(subject string, data []byte) {
	targets := []string{}
	if len(ENV.Config.Duplicate.Remote) > 1 {
		targets = append(targets, ENV.Config.Duplicate.Remote)
	}
	for _, peer := range ENV.Config.Duplicate.Peers {
		if len(peer) > 1 {
			targets = append(targets, peer)
		}
	}
	for _, target := range targets {
		go postToHost(target, subject, data)
	}
}

// postToHostMaxAttempts/postToHostBaseBackoff: this is the one push leg
// that's genuinely capable of being reliable (unlike UDP broadcast, which
// is inherently best-effort at the protocol level) - it was previously a
// single, un-retried attempt with no client timeout at all, so one transient
// failure (the peer momentarily busy, a brief network blip, a half-open
// connection that never errors or completes) permanently dropped that
// close for that one peer, with nothing else re-attempting delivery until
// the next periodic reconciliation tick. Confirmed live: an order closed
// on its pass terminal intermittently never reached a linked sub-screen.
// Retrying a handful of times with a short backoff turns a one-off blip
// into a non-event; reconcileWithPeers (see reconcile.go) remains the
// durable backstop for a peer that's down for longer than these retries
// can bridge.
const postToHostMaxAttempts = 3
const postToHostBaseBackoff = 300 * time.Millisecond

func postToHost(host string, subject string, data []byte) {
	url := "http://" + peerHostPort(host) + "/" + subject + "/"
	client := &http.Client{Timeout: 3 * time.Second}
	backoff := postToHostBaseBackoff
	var lastErr error
	for attempt := 1; attempt <= postToHostMaxAttempts; attempt++ {
		// A fresh request each attempt: the previous attempt's
		// bytes.Buffer is already fully read (consumed) by the time
		// client.Do returns, successfully or not.
		request, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
		if err != nil {
			fmt.Println(err)
			return
		}
		request.Header.Set("Content-Type", "application/json; charset=UTF-8")
		response, err := client.Do(request)
		if err == nil {
			response.Body.Close()
			return
		}
		lastErr = err
		if attempt < postToHostMaxAttempts {
			time.Sleep(backoff)
			backoff *= 2
		}
	}
	fmt.Println("peer notify failed after retries: " + url)
	fmt.Println(lastErr)
}

func sendIfCallBackPOST(subject string, data []byte) {
	if len(ENV.Config.Duplicate.Remote) > 1 {
		fmt.Println(string(data))
		fmt.Println("REMOTE : " + subject)
		var URL = "http://" + peerHostPort(ENV.Config.Duplicate.Remote) + "/" + subject + "/"
		fmt.Println("REMOTE : " + URL)
		request, error := http.NewRequest("POST", URL, bytes.NewBuffer(data))
		request.Header.Set("Content-Type", "application/json; charset=UTF-8")
		if error != nil {
			fmt.Println(error)
		}
		client := &http.Client{}
		response, error := client.Do(request)
		if error != nil {
			fmt.Println(error)
		}
		defer response.Body.Close()

		fmt.Println("response Status:", response.Status)
		fmt.Println("response Headers:", response.Header)
		body, _ := ioutil.ReadAll(response.Body)
		fmt.Println("response Body:", string(body))
	}
}

func checkConnections() {
	//fmt.Println("Checking Connections")

	if NC == nil || !NC.IsConnected() {
		fmt.Println("Local NATS is Dead , Reconnecting")
		initLocalConnection()
	}

	if NCglobal == nil || !NCglobal.IsConnected() {
		fmt.Println("Global NATS is Dead , Reconnecting")
		initGlobalConnection()
	}

	initDuplicateConnections()
}
func initDuplicateConnections() {
	if ENV.Config.Duplicate.Enableduplication {
		fmt.Println("Screen Duplication is Enabled")
		if len(ENV.Config.Duplicate.Screen1) > 1 {
			fmt.Println("Screen1 Duplication")
			if NC1 != nil && NC1.IsConnected() {
				fmt.Println("Screen1 Client Already Connected")
			} else {
				fmt.Println(ENV.Config.Duplicate.Screen1)
				NC1, _ = helper.KDSNatsLocalConnectOverride("MASTER-"+ENV.Config.Client.Serial, ENV.Config.Duplicate.Screen1)
				//NC1, _ = nats.Connect("nats://"+ENV.Config.Duplicate.Screen1+":4222", nats.Timeout(10*time.Second))
				if NC1 != nil && NC1.IsConnected() {
					fmt.Println("Screen1 Client Now Connected")
				} else {
					fmt.Println("Screen1 Client Not Connected")
				}
			}
		}
		if len(ENV.Config.Duplicate.Screen2) > 1 {
			fmt.Println("Screen2 Duplication")
			if NC2 != nil && NC2.IsConnected() {
				fmt.Println("Screen2 Client Already Connected")
			} else {
				NC2, _ = helper.KDSNatsLocalConnectOverride("MASTER-"+ENV.Config.Client.Serial, ENV.Config.Duplicate.Screen2)
				//NC1, _ = nats.Connect("nats://"+ENV.Config.Duplicate.Screen1+":4222", nats.Timeout(10*time.Second))
				if NC2 != nil && NC2.IsConnected() {
					fmt.Println("Screen2 Client Now Connected")
				}
			}
		}
	}
}
func initLocalConnection() {
	NC, conerror = helper.KDSNatsLocalConnectPort("KDSWEB-"+ENV.Config.Client.Serial, natsLocalPort)
	kdshelper.HandelGlobalError(conerror, kdshelper.GetSentryServices().KDSWEBSERVER)
	if NC != nil && NC.IsConnected() {
		fmt.Println("Subscribing Local")
		NC.Subscribe("neworder", func(m *nats.Msg) {
			notifyKDSOrders()
		})
		NC.Subscribe("getorders", func(m *nats.Msg) {
			natsOpenOrders(m)
		})
		NC.Subscribe("pastorders", func(m *nats.Msg) {
			natsPastOrdersRequest(m)
		})
		NC.Subscribe("getsingleorder", func(m *nats.Msg) {
			natsSingleOrder(m)
		})
		NC.Subscribe("sub-order-close", func(m *nats.Msg) {
			// This subscription gets one dedicated delivery goroutine from
			// the NATS client, shared by every close-order event site-wide -
			// natsCloseOrder does real synchronous work per call (DB writes,
			// fan-out to peers/UDP, a full notifyKDSOrders() board refetch),
			// so two closes landing close together used to queue one behind
			// the other's entire processing time instead of running
			// concurrently, same as every other handler in this file already
			// does via net/http's goroutine-per-request. Now safe to run
			// concurrently: the SQLite busy_timeout pragma (helper.go) means
			// a write collision waits instead of silently failing.
			go natsCloseOrder(m)
		})
		NC.Subscribe("sub-order-recall", func(m *nats.Msg) {
			natsRecallOrder(m)
		})
		NC.Subscribe("DUPOPENITEM", func(m *nats.Msg) {
			fmt.Println("DUPOPENITEM")
			var item kdsmodels.KDSItems
			err := json.Unmarshal(m.Data, &item)
			fmt.Println(err)
			if item.ID > 0 {
				//fmt.Println(item.Reference)
				orderhere := getDatabaseOrderFromReference(item.Reference)
				if orderhere.ID > 0 {
					existing := getDatabaseItemByOccurrence(orderhere.ID, item.Name, item.Qty, item.Occurrence)
					if existing.ID > 0 {
						openTheItemDB(existing.ID)
					} else {
						fmt.Println("Invalid Item")
					}
					notifyKDSOrders()
				} else {
					fmt.Println("Invalid Order")
				}
			} else {
				fmt.Println("Invalid Message")
			}
		})
		NC.Subscribe("DUPCLOSEORDER", func(m *nats.Msg) {
			fmt.Println("DUPCLOSEORDER")
			var order kdsmodels.KDSOrders
			err := json.Unmarshal(m.Data, &order)
			fmt.Println(err)
			kdshelper.HandelGlobalError(err, kdshelper.GetSentryServices().KDSWEBSERVER)
			if order.ID > 0 {
				orderhere := getDatabaseOrderFromReference(order.Reference)
				if orderhere.ID > 0 {
					fmt.Println(orderhere.ID)
					// Unconditional close, not screen-filtered by this
					// terminal's own name - see applyCloseOrderRemote in
					// close_order.go for why closeTheOrderDB(id, screenname)
					// is wrong here.
					orders := closeOrderDatabase(orderhere.ID)
					err1 := errors.New("closed dupe order")
					kdshelper.HandelGlobalError(err1, kdshelper.GetSentryServices().KDSWEBSERVER)
					notifyKDSOrders()
					notifyPrepTime(ENV.Config.Client.Terminal)
					js, _ := json.Marshal(&orders)
					notifyEvent("CLOSEDORDER", string(js))
					natsReload()
				} else {
					fmt.Println("Invalid Order")
					err1 := errors.New("invalid local order id")
					kdshelper.HandelGlobalError(err1, kdshelper.GetSentryServices().KDSWEBSERVER)
				}
			} else {
				err1 := errors.New("invalid order id")
				kdshelper.HandelGlobalError(err1, kdshelper.GetSentryServices().KDSWEBSERVER)
				fmt.Println("Invalid Message")
			}
		})
		NC.Subscribe("DUPCLOSEORDERSCREEN", func(m *nats.Msg) {
			fmt.Println("DUPCLOSEORDERSCREEN")
			var closemsg kdsmodels.KDSOrderScreenClose
			err := json.Unmarshal(m.Data, &closemsg)
			if err == nil {
				applyCloseOrderScreenRemote(closemsg)
			}
		})
		NC.Subscribe("DUPSCREENPASSASSIGNMENT", func(m *nats.Msg) {
			fmt.Println("DUPSCREENPASSASSIGNMENT")
			var assignment kdsmodels.KDSScreenPassAssignment
			err := json.Unmarshal(m.Data, &assignment)
			if err == nil {
				upsertScreenPassAssignment(assignment.Screen, assignment.Passterminal)
			}
		})
		NC.Subscribe("DUPPASSTERMINALANNOUNCE", func(m *nats.Msg) {
			fmt.Println("DUPPASSTERMINALANNOUNCE")
			var announce kdsmodels.KDSPassTerminalAnnounce
			err := json.Unmarshal(m.Data, &announce)
			if err == nil {
				upsertPassTerminalAnnounce(announce.Name)
			}
		})
		// DUPDISCOVERPING is the screen1/screen2 NATS-mirror leg of a
		// "Rediscover Pass Terminals" request (see
		// requestPassTerminalRediscovery in pass_terminal_assignment.go) -
		// re-announce ourselves right now instead of waiting for the next
		// periodic 2-minute tick. No payload to decode; receiving this at
		// all is the whole signal.
		NC.Subscribe("DUPDISCOVERPING", func(m *nats.Msg) {
			fmt.Println("DUPDISCOVERPING")
			asyncAnnouncePassTerminalInfo()
		})
		NC.Subscribe("DUPCLOSEITEM", func(m *nats.Msg) {
			fmt.Println("DUPCLOSEITEM")
			var item kdsmodels.KDSItems
			err := json.Unmarshal(m.Data, &item)
			fmt.Println(err)
			if item.ID > 0 {
				orderhere := getDatabaseOrderFromReference(item.Reference)
				if orderhere.ID > 0 {
					existing := getDatabaseItemByOccurrence(orderhere.ID, item.Name, item.Qty, item.Occurrence)
					if existing.ID > 0 {
						closeTheItemDB(existing.ID)
					} else {
						fmt.Println("Invalid Item")
					}
					notifyKDSOrders()
					//natsReload()
				} else {
					fmt.Println("Invalid Order")
				}
			} else {
				fmt.Println("Invalid Message")
			}
		})
		NC.Subscribe("DUPSTARTPRODUCTION", func(m *nats.Msg) {
			fmt.Println("DUPSTARTPRODUCTION")
			var item kdsmodels.KDSItems
			err := json.Unmarshal(m.Data, &item)
			fmt.Println(err)
			if item.ID > 0 {
				orderhere := getDatabaseOrderFromReference(item.Reference)
				if orderhere.ID > 0 {
					existing := getDatabaseItemByOccurrence(orderhere.ID, item.Name, item.Qty, item.Occurrence)
					if existing.ID > 0 {
						now := getTimeSQL()
						existing.InProduction = true
						existing.ProductionStartedAt = &now
						existing.ProductionEndedAt = nil
						ENV.DB.Save(&existing)
					} else {
						fmt.Println("Invalid Item")
					}
					notifyKDSOrders()
				} else {
					fmt.Println("Invalid Order")
				}
			} else {
				fmt.Println("Invalid Message")
			}
		})
		NC.Subscribe("DUPSENDTOTABLE", func(m *nats.Msg) {
			fmt.Println("DUPSENDTOTABLE")
			var item kdsmodels.KDSItems
			err := json.Unmarshal(m.Data, &item)
			fmt.Println(err)
			if item.ID > 0 {
				orderhere := getDatabaseOrderFromReference(item.Reference)
				if orderhere.ID > 0 {
					existing := getDatabaseItemByOccurrence(orderhere.ID, item.Name, item.Qty, item.Occurrence)
					if existing.ID > 0 {
						now := getTimeSQL()
						existing.SentToTable = true
						existing.SentToTableAt = &now
						ENV.DB.Save(&existing)
					} else {
						fmt.Println("Invalid Item")
					}
					notifyKDSOrders()
				} else {
					fmt.Println("Invalid Order")
				}
			} else {
				fmt.Println("Invalid Message")
			}
		})
		NC.Subscribe("DUPCLEARPRODUCTION", func(m *nats.Msg) {
			fmt.Println("DUPCLEARPRODUCTION")
			var item kdsmodels.KDSItems
			err := json.Unmarshal(m.Data, &item)
			fmt.Println(err)
			if item.ID > 0 {
				orderhere := getDatabaseOrderFromReference(item.Reference)
				if orderhere.ID > 0 {
					existing := getDatabaseItemByOccurrence(orderhere.ID, item.Name, item.Qty, item.Occurrence)
					if existing.ID > 0 {
						now := getTimeSQL()
						existing.InProduction = false
						existing.ProductionEndedAt = &now
						ENV.DB.Save(&existing)
					} else {
						fmt.Println("Invalid Item")
					}
					notifyKDSOrders()
				} else {
					fmt.Println("Invalid Order")
				}
			} else {
				fmt.Println("Invalid Message")
			}
		})
	}
}
func initGlobalConnection() {
	NCglobal, conerror = helper.KDSNatsRemoteConnect("KDSWEB-" + ENV.Config.Client.Serial)
	kdshelper.HandelGlobalError(conerror, kdshelper.GetSentryServices().KDSWEBSERVER)

	if NCglobal != nil && NCglobal.IsConnected() {
		fmt.Println("Subscribing Global")
		// Requests Fire-And-Wait from KDS Terminals
		fmt.Println("Subscribing to : " + "kds.terminal." + config.Client.Serial)
		NCglobal.Subscribe("kds.terminal."+config.Client.Serial, func(m *nats.Msg) {
			kdsmessage := kdsmodels.KDSTerminalMessage{}
			err := json.Unmarshal(m.Data, &kdsmessage)
			kdshelper.HandelGlobalError(err, kdshelper.GetSentryServices().KDSWEBSERVER)
			switch kdsmessage.Type {
			case "SEND_DATA":
				//give the terminal its config
				kdsterminal := kdsactions.KDSGetTerminalFromSerial(config.Client.Serial)
				if kdsterminal.ID > 0 {
					orders := getTodaysDatabaseOrdersExport(config)
					js, _ := json.Marshal(orders)
					kdsmessage.Type = "ACK"
					kdsmessage.Message = "OK"
					kdsmessage.TerminalID = kdsterminal.ID
					returnmessage, err := json.Marshal(kdsmessage)
					if err != nil {
						fmt.Println(err)
					}
					m.Respond(returnmessage)
					kdsmessage.Type = "DATA_UPLOAD"
					kdsmessage.Message = string(js)
					kdsmessage.HWID = config.Client.Serial
					returnmessage, _ = json.Marshal(kdsmessage)
					NCglobal.Publish("system.messages", returnmessage)
				}
				fmt.Println("SEND_DATA DONE")
			case "SEND_DATA_ALL":
				//give the terminal its config
				kdsterminal := kdsactions.KDSGetTerminalFromSerial(config.Client.Serial)
				if kdsterminal.ID > 0 {
					orders := getAllDatabaseOrdersExport(config)
					js, _ := json.Marshal(orders)
					kdsmessage.Type = "ACK"
					kdsmessage.Message = "OK"
					kdsmessage.TerminalID = kdsterminal.ID
					returnmessage, err := json.Marshal(kdsmessage)
					kdshelper.HandelGlobalError(err, kdshelper.GetSentryServices().KDSWEBSERVER)
					m.Respond(returnmessage)
					kdsmessage.Type = "DATA_UPLOAD"
					kdsmessage.Message = string(js)
					kdsmessage.HWID = config.Client.Serial
					returnmessage, _ = json.Marshal(kdsmessage)
					NCglobal.Publish("system.messages", returnmessage)
				}
				fmt.Println("SEND_DATA DONE")
			case "SEND_DATA_RANGE":
				//give the terminal its config
				kdsterminal := kdsactions.KDSGetTerminalFromSerial(config.Client.Serial)
				if kdsterminal.ID > 0 {
					RangeDate := RangeData{}
					err := json.Unmarshal([]byte(kdsmessage.Message), &RangeDate)
					kdshelper.HandelGlobalError(err, kdshelper.GetSentryServices().KDSWEBSERVER)
					orders := getRangeDatabaseOrdersExport(config, RangeDate.Start, RangeDate.End)
					fmt.Printf("Sending %d Of Orders", len(orders))
					kdsmessage.Type = "ACK"
					kdsmessage.Message = "OK"
					kdsmessage.TerminalID = kdsterminal.ID
					returnmessage, err := json.Marshal(kdsmessage)
					kdshelper.HandelGlobalError(err, kdshelper.GetSentryServices().KDSWEBSERVER)
					m.Respond(returnmessage)
					kdsmessage.Type = "DATA_UPLOAD"
					kdsmessage.HWID = config.Client.Serial
					sendMultipleOrders(orders, kdsmessage)
				}
				fmt.Println("SEND_DATA DONE")
			default:
				fmt.Println("DONE")
			}
		})
	}
}
func initConnections() {
	initLocalConnection()
	initGlobalConnection()
	initDuplicateConnections()
}

func sendMultipleOrders(orders []kdsmodels.KDSExportOrders, kdsmessage kdsmodels.KDSTerminalMessage) {
	k := len(orders)
	p := 0
	var returnmessage []byte
	for k > p {
		var js []byte
		if k-p >= 10 {
			js, _ = json.Marshal(orders[p : p+10])
			fmt.Println("SENDING 10 ORDERS")
		} else {
			js, _ = json.Marshal(orders[p : p+(k-p)])
			fmt.Println("SENDING " + helper.IntToString(k-p) + " ORDERS")
		}
		kdsmessage.Message = string(js)
		returnmessage, _ = json.Marshal(kdsmessage)
		helper.KDSNATSPublishGlobal(NCglobal, "system.messages", returnmessage)
		p = p + 10
	}
}

func SendTodaysOrders() {
	fmt.Println("Sending Todays Orders")
	kdsterminal := kdsactions.KDSGetTerminalFromSerial(config.Client.Serial)
	if kdsterminal.ID > 0 {
		kdsmessage := kdsmodels.KDSTerminalMessage{}
		orders := getTodaysDatabaseOrdersExport(config)
		kdsmessage.TerminalID = kdsterminal.ID
		kdsmessage.Type = "DATA_UPLOAD"
		kdsmessage.HWID = config.Client.Serial
		sendMultipleOrders(orders, kdsmessage)
	}
}

func UpdatedData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("OK"))
	notifyKDSOrders()
}

func notifyPrepTime(screen string) {
	average := getAveragePrepPerScreen(config, screen)
	var wspacket kdsmodels.KDSWSPacket
	wspacket.Data = average
	wspacket.MessageType = "preptime"
	js, _ := json.Marshal(wspacket)
	NC.Publish("updates", js)
}
func notifyPrepTimeWithoutScreen() {
	average := getAveragePrep(config)
	var wspacket kdsmodels.KDSWSPacket
	wspacket.Data = average
	wspacket.MessageType = "preptime"
	js, _ := json.Marshal(wspacket)
	NC.Publish("updates", js)
}
func natsReload() {
	showDebug("natsReload")
	var wspacket kdsmodels.KDSWSPacket
	wspacket.Data = "[]"
	wspacket.MessageType = "reload"
	js, _ := json.Marshal(wspacket)
	NC.Publish("updates", js)
}
func notifyKDSOrders() {
	//fmt.Println("notifyKDSOrders")
	showDebug("notifyKDSOrders")
	orders := getDatabaseOrders(config)
	var wspacket kdsmodels.KDSWSPacket
	wspacket.Data = orders
	wspacket.MessageType = "update"
	js, _ := json.Marshal(wspacket)
	NC.Publish("updates", js)
}

func notifyEvent(event string, msg string) {
	kdsmessage := kdsmodels.KDSTerminalMessage{}
	kdsmessage.TerminalID = 0
	kdsmessage.Type = event
	kdsmessage.Message = msg
	js, _ := json.Marshal(kdsmessage)
	NC.Publish("events", js)
}

func natsPastOrdersRequest(m *nats.Msg) {
	showDebug("notifyPastOrders")
	orders := getCompletedOrders()
	var wspacket kdsmodels.KDSWSPacket
	wspacket.Data = orders
	wspacket.MessageType = "pastorders"
	js, _ := json.Marshal(wspacket)
	//NC.Publish("pastorders", js)
	m.Respond(js)
}
func natsOpenOrders(m *nats.Msg) {
	showDebug("natsOpenOrders")
	orders := getAllDatabaseOrdersKDS()
	var wspacket kdsmodels.KDSWSPacket
	wspacket.Data = orders
	wspacket.MessageType = "update"
	js, _ := json.Marshal(wspacket)
	m.Respond(js)
	notifyKDSOrders()
}
func natsSingleOrder(m *nats.Msg) {
	showDebug("natsSingleOrder")
	kdsmessage := kdsmodels.KDSTerminalMessage{}
	err := json.Unmarshal(m.Data, &kdsmessage)
	kdshelper.HandelGlobalError(err, kdshelper.GetSentryServices().KDSWEBSERVER)
	order := getSingleDatabaseOrdersKDS(helper.StringToInt(kdsmessage.Message))
	var wspacket kdsmodels.KDSWSPacket
	wspacket.Data = order
	wspacket.MessageType = "single"
	js, _ := json.Marshal(wspacket)
	m.Respond(js)
}
// natsCloseOrder handles a Close Order press from this or a duplicated
// terminal (subject sub-order-close). A non-pass terminal is scoped to its
// own screen only (closeOrderScreenDatabase) - a Drinks screen's own Close
// Order press has no business touching Kitchen's food items.
//
// A pass terminal is different: bumping is a deliberate "this section's
// whole ticket is done, force it closed" action, not conditional on every
// sub-terminal having individually marked its own items complete first.
// This previously called closeOrderDatabase directly, which completed
// every item on the ENTIRE order regardless of screen - meaning a Kitchen
// pass screen's bump would also force-close a Bar drink on the same order,
// exactly the cross-terminal bleed pass mode is supposed to avoid.
// closeOrderForPassScreen fixes that: it force-closes this pass terminal's
// own section (its own screen, plus every screen assigned to it via
// KDSScreens.AssignedPassTerminal / that section's "Pass Terminal" config
// dropdown) without requiring individual item status, but stops there -
// it never reaches into a genuinely separate terminal like Bar.
func natsCloseOrder(m *nats.Msg) {
	showDebug("natsCloseOrder")
	kdsmessage := kdsmodels.KDSTerminalMessage{}
	err := json.Unmarshal(m.Data, &kdsmessage)
	kdshelper.HandelGlobalError(err, kdshelper.GetSentryServices().KDSWEBSERVER)
	orderid := helper.StringToInt(kdsmessage.Message)
	var order kdsmodels.KDSOrders
	if kdsmessage.Passmode {
		order = closeOrderForPassScreen(orderid, kdsmessage.Screen, true)
	} else {
		order = closeOrderScreenDatabase(orderid, kdsmessage.Screen, true)
	}
	var wspacket kdsmodels.KDSWSPacket
	wspacket.Data = order
	wspacket.MessageType = "closed"
	js, _ := json.Marshal(wspacket)
	closemsg := kdsmodels.KDSOrderScreenClose{Reference: order.Reference, Screen: kdsmessage.Screen, Passmode: kdsmessage.Passmode}
	json1, _ := json.Marshal(&closemsg)
	sendIfNeeded("DUPCLOSEORDERSCREEN", json1)
	sendToPeers("closeorderscreenremote", json1)
	broadcastUDP("closeorderscreenremote", json1)
	// Fast-follow safety net: postToHost's own retries (see nat_coms.go)
	// now bridge a one-off transient failure, and UDP is inherently
	// best-effort at the protocol level regardless - but if every leg
	// still misses a peer (e.g. that peer was rebooting at this exact
	// moment), nothing before this would re-examine this close again until
	// asyncReconcileWithPeers' own periodic tick, up to 2 minutes later.
	// Scheduling one extra reconcile pass shortly after every local close -
	// not just waiting on the periodic timer - catches that gap in
	// seconds instead of minutes, same safety gating as the periodic one
	// (reconcileWithPeer only ever finalizes a peer's order when that peer
	// is genuinely deferred to this screen as its pass terminal).
	time.AfterFunc(10*time.Second, reconcileWithPeers)
	notifyPrepTimeWithoutScreen()
	m.Respond(js)

	orderjs, _ := json.Marshal(&order)
	notifyEvent("CLOSEDORDER", string(orderjs))

	notifyKDSOrders()
}
func natsRecallOrder(m *nats.Msg) {
	showDebug("natsCloseOrder")
	kdsmessage := kdsmodels.KDSTerminalMessage{}
	err := json.Unmarshal(m.Data, &kdsmessage)
	kdshelper.HandelGlobalError(err, kdshelper.GetSentryServices().KDSWEBSERVER)
	order := recallOrderDatabase(helper.StringToInt(kdsmessage.Message))
	var wspacket kdsmodels.KDSWSPacket
	wspacket.Data = order
	wspacket.MessageType = "recall"
	js, _ := json.Marshal(wspacket)
	m.Respond(js)
}

/*func natsCloseOrder(m *nats.Msg) {
	showDebug("natsCloseOrder")
	kdsmessage := kdsmodels.KDSTerminalMessage{}
	err := json.Unmarshal(m.Data, &kdsmessage)
	errors.HandelGlobalError(err, errors.GetSentryServices().KDSWEBSERVER)
	order := recallOrderDatabase(generic.StringToInt(kdsmessage.Message))
	var wspacket kdsmodels.KDSWSPacket
	wspacket.Data = order
	wspacket.MessageType = "recall"
	js, _ := json.Marshal(wspacket)
	m.Respond(js)
}*/

func addValue(price int) string {
	toReturn := ""
	if price > 0 {
		toReturn = " (R" + helper.IntToString(price) + ")"
	}
	return toReturn
}

func createVoidItems(sallyMainItem kdsmodels.SallyMainItems, existingitem kdsmodels.KDSItems) {
	spew.Dump(sallyMainItem)
	newitem := kdsmodels.KDSItems{}
	newitem.Name = sallyMainItem.Name + addValue(sallyMainItem.Price)
	newitem.Created = helper.UtilgetTimeSQL()
	newitem.KDSOrderID = existingitem.KDSOrderID
	newitem.Qty = int(math.Abs(float64(sallyMainItem.Qty)))
	newitem.Reference = existingitem.Reference
	newitem.Status = "VOID"
	newitem.KDSScreenID = existingitem.KDSScreenID
	newitem = saveSallyItem(newitem)
	if sallyMainItem.VoidReason != nil && len(*sallyMainItem.VoidReason) > 0 {
		newsubitem := kdsmodels.KDSSubitems{}
		newsubitem.Name = *sallyMainItem.VoidReason
		newsubitem.Qty = 1
		newsubitem.KDSItemID = newitem.ID
		newsubitem.Status = "VOID"
		saveSallySubItem(newsubitem)
	}
	for j := range sallyMainItem.SubItems {
		newsubitem := kdsmodels.KDSSubitems{}
		newsubitem.Name = sallyMainItem.SubItems[j].Name + addValue(sallyMainItem.SubItems[j].Price)
		newsubitem.Qty = int(math.Abs(float64(sallyMainItem.SubItems[j].Qty)))
		newsubitem.KDSItemID = newitem.ID
		newsubitem.Status = "VOID"
		saveSallySubItem(newsubitem)
	}
}

func processSallyMainItem(sallyOrder kdsmodels.SallyOrder, orderid int, isneworder bool) {
	// itemNameOccurrenceSoFar/subNameOccurrenceSoFar disambiguate same-named
	// items/subitems within this one incoming Sally round, the same way
	// itemOccurrenceInOrder ranks by local id order - the Sally wire format
	// itself carries no occurrence, so it's derived fresh each round from
	// each item/subitem's own rank among same-named siblings seen so far in
	// THIS payload's own item order. See getItemSally/getSubItemSally.
	itemNameOccurrenceSoFar := map[string]int{}
	for g := range sallyOrder.Items {
		isvoid := false
		if sallyOrder.Items[g].Qty < 0 {
			isvoid = true
		}
		itemNameOccurrenceSoFar[sallyOrder.Items[g].Name]++
		existingitem := getItemSally(orderid, sallyOrder.Items[g], itemNameOccurrenceSoFar[sallyOrder.Items[g].Name])
		if existingitem.ID > 0 && !isneworder {
			createvoiditems := false
			spew.Dump("ITEM EXISTS")
			screen := getScreenSally(sallyOrder.ProductCenter)
			existingitem.Name = sallyOrder.Items[g].Name
			if isvoid && existingitem.Status == "OPEN" {
				existingitem.Qty = existingitem.Qty + sallyOrder.Items[g].Qty
				if existingitem.Qty <= 0 {
					existingitem.Status = "VOID"
					existingitem.Qty = sallyOrder.Items[g].Qty
					spew.Dump("ITEM VOIDED")
				}
				if sallyOrder.Items[g].Qty < 0 && existingitem.Status != "VOID" {
					//this is a void order but there is still some items left
					spew.Dump("PARTIAL VOIDED")
					createvoiditems = true
				}
			}

			existingitem.KDSScreenID = screen.ID
			saveSallyItem(existingitem)
			subNameOccurrenceSoFar := map[string]int{}
			for j := range sallyOrder.Items[g].SubItems {
				subNameOccurrenceSoFar[sallyOrder.Items[g].SubItems[j].Name]++
				existingsubitem := getSubItemSally(existingitem.ID, sallyOrder.Items[g].SubItems[j], subNameOccurrenceSoFar[sallyOrder.Items[g].SubItems[j].Name])
				if existingsubitem.ID > 0 {
					spew.Dump("SUB ITEM EXISTS")
					issubvoid := false
					if sallyOrder.Items[g].SubItems[j].Qty < 0 {
						issubvoid = true
					}
					if issubvoid && existingsubitem.Status == "OPEN" {
						existingsubitem.Qty = existingsubitem.Qty + sallyOrder.Items[g].SubItems[j].Qty
						if existingsubitem.Qty <= 0 {
							existingsubitem.Status = "COMPLETE"
							spew.Dump("SUB ITEM VOIDED")
						} else {
							existingsubitem.Status = existingitem.Status
						}
					} else if issubvoid && existingsubitem.Status == "VOID" {
						spew.Dump("SUB ITEM VOIDED ALREADY")
						existingsubitem.QtyRaw = float64(0)
						existingsubitem.Qty = 0
					} else {
						spew.Dump("SUB ITEM UPDATED")
						existingsubitem.Status = existingitem.Status
						existingsubitem.QtyRaw = float64(sallyOrder.Items[g].SubItems[j].Qty)
						existingsubitem.Qty = sallyOrder.Items[g].SubItems[j].Qty
					}
					existingsubitem.Name = sallyOrder.Items[g].SubItems[j].Name
					saveSallySubItem(existingsubitem)
				} else {
					if sallyOrder.Items[g].SubItems[j].Qty > 0 {
						spew.Dump("SUB ITEM NOT EXISTS")
						newsubitem := kdsmodels.KDSSubitems{}
						newsubitem.Name = sallyOrder.Items[g].SubItems[j].Name
						newsubitem.Qty = sallyOrder.Items[g].SubItems[j].Qty
						newsubitem.KDSItemID = existingitem.ID
						newsubitem.Status = existingitem.Status
						spew.Dump(newsubitem)
						saveSallySubItem(newsubitem)
					}
				}
			}
			if createvoiditems {
				spew.Dump("PARTIAL VOID")
				createVoidItems(sallyOrder.Items[g], existingitem)
			}
		} else {
			subOrderId := 0
			if !isneworder {
				fmt.Println("NEW ITEM OLD ORDER")
				SuborderCounter := NewSubOrder(orderid)

				subOrderId = SuborderCounter + 1
			}
			screen := getScreenSally(sallyOrder.ProductCenter)
			if sallyOrder.Items[g].Course != nil {
				newitem := kdsmodels.KDSItems{}
				newitem.Created = helper.UtilgetTimeSQL()
				newitem.KDSOrderID = orderid
				newitem.Name = "COURSE : " + IntToString(int(*sallyOrder.Items[g].Course))
				newitem.Qty = 1
				newitem.Reference = "COURSE : " + IntToString(int(*sallyOrder.Items[g].Course))
				newitem.Status = "COURSE"
				newitem.KDSScreenID = screen.ID
				newitem = saveSallyItem(newitem)
			}
			if sallyOrder.Items[g].Seat != nil {
				newitem := kdsmodels.KDSItems{}
				newitem.Created = helper.UtilgetTimeSQL()
				newitem.KDSOrderID = orderid
				newitem.Name = "SEAT : " + IntToString(int(*sallyOrder.Items[g].Seat))
				newitem.Qty = 1
				newitem.Reference = "SEAT : " + IntToString(int(*sallyOrder.Items[g].Seat))
				newitem.Status = "COURSE"
				newitem.KDSScreenID = screen.ID
				newitem = saveSallyItem(newitem)
			}
			if sallyOrder.Items[g].Qty > 0 {
				spew.Dump("ITEM NOT EXISTS")
				newitem := kdsmodels.KDSItems{}
				newitem.Created = helper.UtilgetTimeSQL()
				newitem.KDSOrderID = orderid
				newitem.Name = sallyOrder.Items[g].Name
				newitem.Qty = sallyOrder.Items[g].Qty
				newitem.Reference = sallyOrder.Items[g].Sku
				newitem.Status = "OPEN"
				newitem.KDSScreenID = screen.ID
				newitem.SubOrder = subOrderId
				newitem = saveSallyItem(newitem)
				if sallyOrder.Items[g].Note != nil && len(*sallyOrder.Items[g].Note) > 0 {
					newsubitem := kdsmodels.KDSSubitems{}
					newsubitem.Name = *sallyOrder.Items[g].Note
					newsubitem.QtyRaw = 1
					newsubitem.Qty = 1
					newsubitem.KDSItemID = newitem.ID
					newsubitem.Status = newitem.Status
					saveSallySubItem(newsubitem)
				}
				newItemSubNameOccurrenceSoFar := map[string]int{}
				for j := range sallyOrder.Items[g].SubItems {
					newItemSubNameOccurrenceSoFar[sallyOrder.Items[g].SubItems[j].Name]++
					existingsubitem := getSubItemSally(newitem.ID, sallyOrder.Items[g].SubItems[j], newItemSubNameOccurrenceSoFar[sallyOrder.Items[g].SubItems[j].Name])
					if existingsubitem.ID > 0 {
						spew.Dump("SUB ITEM EXISTS")
						existingsubitem.Name = sallyOrder.Items[g].SubItems[j].Name
						existingsubitem.QtyRaw = float64(sallyOrder.Items[g].SubItems[j].Qty)
						existingsubitem.Qty = sallyOrder.Items[g].SubItems[j].Qty
						existingsubitem.Status = newitem.Status
						saveSallySubItem(existingsubitem)
					} else {
						spew.Dump("SUB ITEM NOT EXISTS")
						newsubitem := kdsmodels.KDSSubitems{}
						newsubitem.Name = sallyOrder.Items[g].SubItems[j].Name
						newsubitem.QtyRaw = float64(sallyOrder.Items[g].SubItems[j].Qty)
						newsubitem.Qty = sallyOrder.Items[g].SubItems[j].Qty
						newsubitem.KDSItemID = newitem.ID
						newsubitem.Status = newitem.Status
						saveSallySubItem(newsubitem)
					}
				}
			}
		}
	}
}

func ProcessSallyPing(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

func ProcessSallyOrder(w http.ResponseWriter, r *http.Request) {
	var sallyOrder kdsmodels.SallyOrder
	err := json.NewDecoder(r.Body).Decode(&sallyOrder)
	if err != nil {
		w.Write([]byte(err.Error()))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	existingorder := getDatabaseOrderOnReference(config, sallyOrder.Id)
	// D01: a table whose order is fully sent but never explicitly bumped is
	// a finished meal nobody's gotten around to closing - not a still-open
	// tab. An update carrying new items for that same reference (e.g. the
	// POS reusing it for the table's next sitting) is a genuinely new
	// order, not more of the old one. See kds-poller's own copy of this
	// same check (engine.go) for the full reasoning. Scoped to updates that
	// actually carry new items, same as the poller side - an update with no
	// items isn't trying to start anything new.
	if existingorder.ID > 0 && len(sallyOrder.Items) > 0 && orderIsFullySentButOpen(existingorder.ID) {
		existingorder = kdsmodels.KDSOrders{}
	}
	if existingorder.ID > 0 {
		spew.Dump("EXISTING")
		existingorder.Reference = sallyOrder.Id
		existingorder.Table = sallyOrder.TableName
		existingorder.Waiter = sallyOrder.WaiterName
		existingorder.Status = "OPEN"
		if sallyOrder.Profile == nil {
			existingorder.Profile = sallyOrder.ProductCenter
			fmt.Println("EMPTY Profile")
		} else {
			existingorder.Profile = string(*sallyOrder.Profile)
			fmt.Println("NEW Profile")
			fmt.Println(string(*sallyOrder.Profile))
		}

		if sallyOrder.ConsumerName != nil {
			existingorder.Consumer = string(*sallyOrder.ConsumerName)
		}
		existingorder = saveSallyOrder(existingorder)
		processSallyMainItem(sallyOrder, existingorder.ID, false)
	} else {
		spew.Dump("NEW")
		neworder := kdsmodels.KDSOrders{}
		spew.Dump(NewOrderForToday())
		ordercounter := NewOrderForToday() + 1
		ordercounter++
		neworder.OrderFortoday = ordercounter
		neworder.Reference = sallyOrder.Id
		neworder.Table = sallyOrder.TableName
		neworder.Waiter = sallyOrder.WaiterName
		neworder.Status = "OPEN"
		if sallyOrder.Profile == nil {
			neworder.Profile = sallyOrder.ProductCenter
		} else {
			neworder.Profile = string(*sallyOrder.Profile)
		}
		neworder.Created = helper.UtilgetTimeSQL()
		neworder.PassStatus = "OPEN"
		if sallyOrder.ConsumerName != nil {
			neworder.Consumer = string(*sallyOrder.ConsumerName)
		}

		neworder = saveSallyOrder(neworder)
		processSallyMainItem(sallyOrder, neworder.ID, true)

		// Tell every other terminal a new order exists, carrying this
		// terminal's own OrderFortoday - so a terminal that never ingests
		// tickets itself (an Order Display Screen) shows the exact same
		// "Chk N" number this terminal does, instead of racing its own
		// independent daily counter. See applyNewOrderRemote.
		fullorder := getDatabaseOrder(config, IntToString(neworder.ID))
		fullorder.Reference = neworder.Reference
		orderjs, marshalErr := json.Marshal(&fullorder)
		if marshalErr == nil {
			sendIfNeeded("DUPNEWORDER", orderjs)
			sendToPeers("neworderremote", orderjs)
			broadcastUDP("neworderremote", orderjs)
		}
	}
	//spew.Dump(sallyOrder)
	//w.Write([]byte("OK"))
	notifyKDSOrders()
	UpdatedData(w, r)
}
