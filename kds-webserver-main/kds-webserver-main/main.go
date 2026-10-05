package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/reverent911/kds-golang-lib/pkg/generic/errors"
	"github.com/reverent911/kds-golang-lib/pkg/kds/kdshelper"
	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
)

var ENV *kdsmodels.KDSEnv
var natsLocalPort = 4222
var debug = false
var DBLocation string
var config kdsmodels.KDSConfig
// configLocation is the config file's own path, kept as a package-level var
// (rather than main()'s local one) so announceOwnScreenPassAssignment/
// announceAsPassTerminal (see pass_terminal_assignment.go) can re-read it
// fresh from disk on demand - ENV.Config is only ever populated once, at
// this process's own boot, so a Client Config change made after that (e.g.
// setting Pass Terminal on a section screen) would otherwise never reach
// the announce functions until this whole service restarts.
var configLocation string

func openCompletedOrder(orderid string) {
	order := kdsmodels.KDSOrders{}
	ENV.DB.Where("id = ?", orderid).Where("DATE(created) = DATE(CURRENT_TIMESTAMP, 'localtime')").First(&order)
	if order.ID > 0 {
		items := []kdsmodels.KDSItems{}
		ENV.DB.Where("kds_order_id = ?", orderid).Where("status = ?", "COMPLETE").Find(&items)
		for i := range items {
			items[i].Status = "OPEN"
			items[i].Created = fixDateTime(items[i].Created)
			ENV.DB.Save(&items[i])
		}
		//order.Completed = &(fixDateTime(order.Completed))
		//order.Created = fixDateTime(order.Created)
		order.Status = "OPEN"
		order.PassStatus = "OPEN"
		ENV.DB.Save(&order)
	}
}

// SCREEN WEBSOCKET CODE -----------------------------------------------------------------------------------------------------------

/* get screen database orders */

func getAveragePrepPerScreen(config kdsmodels.KDSConfig, screenname string) kdsmodels.PrepTime {
	screen := kdsmodels.KDSScreens{}
	avg := kdsmodels.PrepTime{}
	ENV.DB.Where("LOWER(name) = LOWER(?)", screenname).First(&screen)
	if screen.ID > 0 {
		ENV.DB.Raw("select ROUND(AVG(prep_mins),0) as avgmins , ROUND(AVG(prep_secs),0) as avgsecs from kds_items where status = 'COMPLETE' and DATE(created) = DATE(current_timestamp, 'localtime') and kds_screen_id = ?", screen.ID).Scan(&avg)
	}
	return avg
}

func getAveragePrep(config kdsmodels.KDSConfig) kdsmodels.PrepTime {
	avg := kdsmodels.PrepTime{}
	ENV.DB.Raw("select ROUND(AVG(prep_mins),0) as avgmins , ROUND(AVG(prep_secs),0) as avgsecs from kds_items where status = 'COMPLETE' and DATE(created) = DATE(current_timestamp, 'localtime')").Scan(&avg)
	return avg
}

/* ========= ASYNC  ======= */

func asyncUploadOrders() {
	for range time.Tick(time.Minute * 360) {
		SendTodaysOrders()
	}
}

func asyncCheckConnections() {
	for range time.Tick(time.Second * 60) {
		checkConnections()
	}
}

func asyncAnnouncePassTerminal() {
	for range time.Tick(time.Minute * 2) {
		asyncAnnouncePassTerminalInfo()
	}
}

/* ========= END ASYNC ========= */

func main() {
	baseLocation := flag.String("base", "/opt/kds/", "Base KDS Location")
	// Was hardcoded (":2020"), so two instances could never coexist on the
	// same machine - production only ever runs one webserver per terminal,
	// so this changes nothing there (defaults to :2020 exactly as before);
	// it exists purely so a second instance can be launched locally (e.g.
	// for testing peer sync against a real second terminal) without a port
	// clash. See peerHostPort (nat_coms.go) for the matching change on the
	// OUTGOING side - a peer address can now optionally carry its own port
	// too, for exactly this same local-testing case.
	httpPort := flag.Int("port", 2020, "HTTP listen port")
	// See KDSNatsLocalConnectPort's own doc comment - same local-testing
	// reasoning as -port above, for this terminal's own local NATS
	// connection (production always uses the default 4222).
	natsPort := flag.Int("natsport", 4222, "Local NATS port")
	flag.Parse()
	natsLocalPort = *natsPort
	// Assigns the package-level configLocation/DBLocation directly (= not
	// :=, which would instead declare new locals shadowing them) -
	// announceOwnScreenPassAssignment/announceAsPassTerminal need the real
	// configLocation to re-read the config file fresh on demand (see its
	// own declaration above for why).
	_, configLocation, DBLocation = buildConfig(*baseLocation)

	app, err := errors.CreateNewRelic("KDS Webserver")
	kdshelper.HandelGlobalError(err, kdshelper.GetSentryServices().KDSWEBSERVER)

	ENV = InitConfSQLite(configLocation, DBLocation)
	migrateProductionColumns()
	migrateOrphanedBlankScreenItems()

	go initConnections()
	go startUDPDiscovery()
	go asyncAnnouncePassTerminalInfo()

	r := mux.NewRouter()
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/getsingleorder/{id}", getSingleOrderHandler))
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/getsingledborder/{id}", getSingleDBOrderHandler))
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/getcompletedorders/{id}", GetcompletedHandler))
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/recallorder/{id}/{screen}/{passmode}", recallorderHandler))
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/getconfig", GetconfigHandler))
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/closeorder/{id}/{screen}", closeOrdersHandler))
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/closeitem/{id}", closeItemsHandler))
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/openitem/{id}", openItemsHandler))
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/startproduction/{id}", startProductionHandler))
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/clearproduction/{id}", clearProductionHandler))
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/senditemtotable/{id}", sendItemToTableHandler))
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/updated", UpdatedData))

	//Remote Functions
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/closeitemremote/", closeRemoteItemHandler))
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/openitemremote/", openItemsRemoteHandler))
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/closeremote/{id}", closeRemoteOrdersHandler))
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/closeorderremote/", closeOrderRemoteHandler))
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/closeorderscreenremote/", closeOrderScreenRemoteHandler))
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/recallorderscreenremote/", recallOrderScreenRemoteHandler))
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/screenpassassignmentremote/", screenPassAssignmentRemoteHandler))
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/passterminalannounceremote/", passTerminalAnnounceRemoteHandler))
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/getpassterminals", getPassTerminalsHandler))
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/getknownterminals", getKnownTerminalsHandler))
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/rediscoverpassterminals", rediscoverPassTerminalsHandler))
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/repairbrokenorders", repairBrokenOrdersHandler))
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/discoverpingremote/", discoverPingRemoteHandler))
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/startproductionremote/", startProductionRemoteHandler))
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/clearproductionremote/", clearProductionRemoteHandler))
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/sendtotableremote/", sendToTableRemoteHandler))
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/neworderremote/", newOrderRemoteHandler))
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/reconcilesnapshot", reconcileSnapshotHandler))

	r.HandleFunc(newrelic.WrapHandleFunc(app, "/sallyorder", ProcessSallyOrder))
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/order", ProcessSallyOrder))
	r.HandleFunc(newrelic.WrapHandleFunc(app, "/ping", ProcessSallyPing))

	fmt.Println("Starting Webserver 1.1")

	srv := &http.Server{
		Handler:      r,
		Addr:         fmt.Sprintf(":%d", *httpPort),
		WriteTimeout: 30 * time.Second,
		ReadTimeout:  30 * time.Second,
	}

	go asyncCheckConnections()

	go asyncUploadOrders()

	go asyncAnnouncePassTerminal()

	go asyncReconcileWithPeers()

	log.Fatal(srv.ListenAndServe())
}
