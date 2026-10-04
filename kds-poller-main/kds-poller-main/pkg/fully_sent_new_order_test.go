package main

import (
	"github.com/glebarez/sqlite"
	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
	"gorm.io/gorm"
	"testing"
)

// D01: a table whose order is fully sent (every real item COMPLETE and
// physically delivered) but never explicitly bumped must have its next
// ticket carrying new items start a genuinely new order, not get silently
// merged into the already-finished one. Mirrors kds-webserver-main's own
// copy of this same test (fully_sent_new_order_test.go) for the raw-ticket
// ingestion path. Reuses setupPollerTestDB from
// engine_control_byte_test.go - same package, same isolated temp-db
// pattern.

const d01Round1Ticket = " * dine in *\n" +
	"table 9\n" +
	"* * kitchen * *\n" +
	"ipad3/sam\n" +
	"10:19 pm a1.1\n" +
	"1 x burger\n" +
	"\x1dvb\x00\n"

const d01Round2Ticket = " * dine in *\n" +
	"table 9\n" +
	"* * kitchen * *\n" +
	"ipad3/sam\n" +
	"10:25 pm a1.1\n" +
	"1 x pizza\n" +
	"\x1dvb\x00\n"

func TestProcessMessage_FullySentOrderStartsFreshOnNextTicketWithItems(t *testing.T) {
	_, dbPath := setupPollerTestDB(t)
	DBLocation = dbPath
	FileLocation = t.TempDir()

	obj := kdsmodels.KDSPollObject{}
	obj.Config = kdsmodels.KDSConfig{}
	ProcessMessage(obj, []byte(d01Round1Ticket), FileLocation, dbPath)

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	defer closeConnection(db)

	var firstOrder kdsmodels.KDSOrders
	if err := db.Where("reference = ?", "a1.1").First(&firstOrder).Error; err != nil {
		t.Fatal(err)
	}
	var firstItem kdsmodels.KDSItems
	if err := db.Where("kds_order_id = ?", firstOrder.ID).First(&firstItem).Error; err != nil {
		t.Fatal(err)
	}
	// Simulate the burger being completed and physically sent to the table
	// via the GUI, without anyone pressing Close Order.
	firstItem.Status = "COMPLETE"
	firstItem.SentToTable = true
	if err := db.Save(&firstItem).Error; err != nil {
		t.Fatal(err)
	}
	if !orderIsFullySentButOpen(db, firstOrder.ID) {
		t.Fatal("expected the order to be recognized as fully sent but open before round 2")
	}

	ProcessMessage(obj, []byte(d01Round2Ticket), FileLocation, dbPath)

	var orders []kdsmodels.KDSOrders
	if err := db.Where("reference = ?", "a1.1").Find(&orders).Error; err != nil {
		t.Fatal(err)
	}
	if len(orders) != 2 {
		t.Fatalf("expected the fully-sent order to be left alone and a new one created, got %d order(s): %+v", len(orders), orders)
	}

	var firstOrderItems, secondOrderItems []kdsmodels.KDSItems
	for _, o := range orders {
		if o.ID == firstOrder.ID {
			db.Where("kds_order_id = ?", o.ID).Find(&firstOrderItems)
		} else {
			db.Where("kds_order_id = ?", o.ID).Find(&secondOrderItems)
		}
	}
	if len(firstOrderItems) != 1 || firstOrderItems[0].Name != "burger" {
		t.Fatalf("the original order must still only have its own burger, got %+v", firstOrderItems)
	}
	if len(secondOrderItems) != 1 || secondOrderItems[0].Name != "pizza" {
		t.Fatalf("the new item must land on the new order, got %+v", secondOrderItems)
	}
}

func TestProcessMessage_StillOpenOrderAppendsNormally(t *testing.T) {
	_, dbPath := setupPollerTestDB(t)
	DBLocation = dbPath
	FileLocation = t.TempDir()

	obj := kdsmodels.KDSPollObject{}
	obj.Config = kdsmodels.KDSConfig{}
	ProcessMessage(obj, []byte(d01Round1Ticket), FileLocation, dbPath)
	// Deliberately NOT marking the burger complete/sent.

	ProcessMessage(obj, []byte(d01Round2Ticket), FileLocation, dbPath)

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	defer closeConnection(db)

	var orders []kdsmodels.KDSOrders
	if err := db.Where("reference = ?", "a1.1").Find(&orders).Error; err != nil {
		t.Fatal(err)
	}
	if len(orders) != 1 {
		t.Fatalf("a still-open order must keep receiving round 2's items, not fork into a second order, got %d orders", len(orders))
	}

	var items []kdsmodels.KDSItems
	if err := db.Where("kds_order_id = ?", orders[0].ID).Find(&items).Error; err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected pizza appended to the same still-open order, got %d items: %+v", len(items), items)
	}
}
