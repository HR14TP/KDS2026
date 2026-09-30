package main

import (
	"log"
	"os"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// CheckExistingOrder ... CheckExistingOrder
func CheckExistingOrder(db *gorm.DB, reference string) kdsmodels.KDSOrders {
	var order kdsmodels.KDSOrders
	db.Where("reference = ?", reference).Where("DATE(kds_orders.created) = DATE(CURRENT_TIMESTAMP)").Where("status =?", "OPEN").First(&order)
	return order
}

// FindOpenOrderAtTable finds whichever order is currently open and sitting
// at the given table, regardless of reference - used as a fallback when a
// transfer ticket's own reference doesn't match any known order (see
// ProcessMessage in engine.go for why that happens). "table" is
// double-quoted because it's a SQL reserved word.
func FindOpenOrderAtTable(db *gorm.DB, table string) kdsmodels.KDSOrders {
	var order kdsmodels.KDSOrders
	db.Where(`"table" = ?`, table).
		Where("status = ?", "OPEN").
		Where("DATE(created) = DATE(CURRENT_TIMESTAMP)").
		Order("id DESC").
		First(&order)
	return order
}

// OpenOrderFamilyIDs returns the given order's own ID plus the IDs of any
// still-open transfer-child tiles spun off from the same original
// reference (identifiable by the "-XFER-" suffix ProcessMessage gives a
// newly-created transfer tile's reference, see engine.go) - used so an
// action addressed by the ORIGINAL ticket reference (e.g. a void, which
// carries no transfer info of its own) can still find an item that has
// since moved to one of those child tiles, instead of only ever looking at
// the original order.
func OpenOrderFamilyIDs(db *gorm.DB, familyReference string, primaryOrderID int) []int {
	ids := []int{primaryOrderID}
	var children []kdsmodels.KDSOrders
	db.Where("reference LIKE ?", familyReference+"-XFER%").
		Where("status = ?", "OPEN").
		Find(&children)
	for _, c := range children {
		ids = append(ids, c.ID)
	}
	return ids
}

// OpenOrderFamilyIDsAtTable is the table-based counterpart to
// OpenOrderFamilyIDs, used when a void ticket's reference doesn't match any
// known order (see ProcessMessage in engine.go) and the only way back to the
// right item is the table name both share. Unlike FindOpenOrderAtTable
// (which just picks whichever open order at that table was created most
// recently), this returns every open order's family at that table, so the
// void match can be decided by which one actually has the named item rather
// than by recency - two separate original orders can each land a transfer
// on the same destination table at the same time, and the one most recently
// created isn't necessarily the one the void is for.
func OpenOrderFamilyIDsAtTable(db *gorm.DB, table string) []int {
	var orders []kdsmodels.KDSOrders
	db.Where(`"table" = ?`, table).
		Where("status = ?", "OPEN").
		Where("DATE(created) = DATE(CURRENT_TIMESTAMP)").
		Find(&orders)
	seen := map[int]bool{}
	var ids []int
	for _, o := range orders {
		for _, fid := range OpenOrderFamilyIDs(db, o.Reference, o.ID) {
			if !seen[fid] {
				seen[fid] = true
				ids = append(ids, fid)
			}
		}
	}
	return ids
}

// FindHashLastHour ... FindHashLastHour
func FindHashLastHour(db *gorm.DB, hash string) bool {
	var fhash kdsmodels.KDSFiles
	db.Raw("SELECT * FROM kds.kds_files where hash = ? and created >= DATE_SUB(NOW(),INTERVAL 1 HOUR) limit 1", hash).Scan(&fhash)
	return fhash.ID > 0
}

// NewOrderForToday ... NewOrderForToday
func NewOrderForToday(db *gorm.DB) int {
	var res kdsmodels.ForToday
	db.Raw("SELECT count(*) as counter from kds_orders where DATE(kds_orders.created) = DATE(CURRENT_TIMESTAMP)").Scan(&res)
	return res.Counter
}

// GetSubOrderForItems ... GetSubOrderForItems
func GetSubOrderForItems(db *gorm.DB, orderid int, screenid int) int {
	var res kdsmodels.ItemSubOrder
	db.Raw("select max(suborder) as counter , count(id) as numoforders from kds_items where kds_order_id = ? and kds_screen_id = ?", orderid, screenid).Scan(&res)
	if res.Numoforders == 0 {
		res.Counter = -1
	}
	return res.Counter
}

// LoadOrCreateScreen ... LoadOrCreateScreen
//
// A blank name (a ticket whose screen marker didn't parse) is deliberately
// left as an empty Name rather than renamed to some fallback - kds-webserver's
// closeOrderForPassTerminal and closeOrderScreenDatabase both always sweep
// up any item stuck on a blank-named screen when their own order is closed
// (see their own comments there), so leaving it blank here is exactly what
// keeps that sweep able to find it. Renaming it to a fixed non-blank string
// would silently break that sweep for every item landing here from that
// point on, since it would only ever match a screen whose name is
// genuinely "", not one deliberately renamed away from it.
//
// The lookup is case-insensitive (LOWER(name) = LOWER(?)) so a ticket
// parsed as "Kitchen" reuses the same screen row as one parsed as
// "kitchen" instead of silently creating a case-variant duplicate that a
// correctly-configured terminal (whatever case it happens to type its own
// screen name in) could then never match against.
func LoadOrCreateScreen(db *gorm.DB, name string) kdsmodels.KDSScreens {
	var screen kdsmodels.KDSScreens
	if name != "" {
		db.Where("LOWER(name) = LOWER(?)", name).Find(&screen)
	} else {
		db.Where("name = ?", name).Find(&screen)
	}
	if screen.ID > 0 {
		return screen
	}
	screen.Name = name
	screen.Port = "hdmi"
	db.Create(&screen)
	return screen

}

// SaveFileHash ... SaveFileHash
func SaveFileHash(db *gorm.DB, hash string, fname string) bool {
	found := FindHashLastHour(db, hash)
	if found {
		return false
	}
	var fhash kdsmodels.KDSFiles
	fhash.Hash = hash
	fhash.Filename = fname
	db.Save(&fhash)
	return true
}

func closeConnection(db *gorm.DB) {
	sqldb, err := db.DB()
	HandleErrorNonFatal(err, nil)
	sqldb.Close()
}

func createSQLITEDatabase(conf kdsmodels.KDSConfig, txn *newrelic.Transaction, dbLocation string) *gorm.DB {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold: time.Second,   // Slow SQL threshold
			LogLevel:      logger.Silent, // Log level
			Colorful:      false,         // Disable color
		},
	)
	// kds-webserver opens this same file independently (its own connection
	// pool, no shared lock) - see the matching comment in kds-webserver's
	// helper.go for why this needs a busy_timeout rather than failing a
	// contended write immediately.
	db, err := gorm.Open(sqlite.Open(dbLocation+"?_pragma=busy_timeout(5000)"), &gorm.Config{
		SkipDefaultTransaction: true,
		Logger:                 newLogger,
	})
	HandleErrorNonFatal(err, txn)
	return db
}
