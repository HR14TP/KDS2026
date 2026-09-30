package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
)

var NatsGlobal *nats.Conn
var Debug bool = true
var FileLocation string
var ConfigLocation string
var DBLocation string
var KDSConfig kdsmodels.KDSConfig

func main() {
	baseLocation := flag.String("base", "/opt/kds/", "Base KDS Location")
	flag.Parse()
	FileLocation, ConfigLocation, DBLocation = buildConfig(*baseLocation)
	fmt.Println("Starting Poller")
	KDSConfig = LoadConfiguration(ConfigLocation)
	fmt.Println("Watching for files on : " + FileLocation)
	go checkConnections(KDSConfig, DBLocation)
	migrate()
	for {
		for range time.Tick(time.Second * 60) {
			checkConnections(KDSConfig, DBLocation)
		}
		select {}
	}
}

func migrate() {
	db := createSQLITEDatabase(KDSConfig, nil, DBLocation)
	db.Exec("ALTER TABLE kds_items ADD qty_raw REAL;COMMIT;")
	db.Exec("ALTER TABLE kds_subitems ADD qty_raw REAL;COMMIT;")
	db.Exec("ALTER TABLE kds_items ADD in_production BOOLEAN DEFAULT 0;COMMIT;")
	db.Exec("ALTER TABLE kds_items ADD production_started_at DATETIME;COMMIT;")
	db.Exec("ALTER TABLE kds_items ADD production_ended_at DATETIME;COMMIT;")
	db.Exec("ALTER TABLE kds_orders ADD transfer_note TEXT;COMMIT;")
	// kds_screens gained this column for the per-section pass-terminal
	// assignment feature (kds-webserver-main) - the poller shares the
	// KDSScreens struct (LoadOrCreateScreen in database.go) so its own
	// local copy of this table needs the same column or every Save/Find
	// against KDSScreens here errors with "no such column".
	db.Exec("ALTER TABLE kds_screens ADD assigned_pass_terminal TEXT;COMMIT;")
}
