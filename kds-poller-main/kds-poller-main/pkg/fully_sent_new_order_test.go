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

// Product decision 2026-10-04: a still-open order's next round of items now
// gets its own fresh tile instead of merging into the original one - see
// the "new round gets its own tile" comment in ProcessMessage (engine.go).
// This replaces the old merge-in-place expectation this test used to
// assert (same ticket fixtures, opposite outcome).
func TestProcessMessage_StillOpenOrderGetsItsOwnNewTileForNextRound(t *testing.T) {
	_, dbPath := setupPollerTestDB(t)
	DBLocation = dbPath
	FileLocation = t.TempDir()

	obj := kdsmodels.KDSPollObject{}
	obj.Config = kdsmodels.KDSConfig{}
	ProcessMessage(obj, []byte(d01Round1Ticket), FileLocation, dbPath)
	// Deliberately NOT marking the burger complete/sent - this order is a
	// perfectly ordinary still-open tab, not the D01 fully-sent case.

	ProcessMessage(obj, []byte(d01Round2Ticket), FileLocation, dbPath)

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	defer closeConnection(db)

	var firstOrder kdsmodels.KDSOrders
	if err := db.Where("reference = ?", "a1.1").First(&firstOrder).Error; err != nil {
		t.Fatal(err)
	}
	var orders []kdsmodels.KDSOrders
	if err := db.Where("reference LIKE ?", "a1.1%").Find(&orders).Error; err != nil {
		t.Fatal(err)
	}
	if len(orders) != 2 {
		t.Fatalf("expected the original order plus a fresh tile for round 2, got %d order(s): %+v", len(orders), orders)
	}

	var firstOrderItems []kdsmodels.KDSItems
	if err := db.Where("kds_order_id = ?", firstOrder.ID).Find(&firstOrderItems).Error; err != nil {
		t.Fatal(err)
	}
	if len(firstOrderItems) != 1 || firstOrderItems[0].Name != "burger" {
		t.Fatalf("the original order must still only have its own burger, untouched by round 2, got %+v", firstOrderItems)
	}

	var roundOrder kdsmodels.KDSOrders
	for _, o := range orders {
		if o.ID != firstOrder.ID {
			roundOrder = o
		}
	}
	if roundOrder.ID == 0 {
		t.Fatal("expected a second order for round 2")
	}
	if roundOrder.Reference != "a1.1-R1" {
		t.Fatalf("expected the new tile's reference derived from the original with a round suffix, got %q", roundOrder.Reference)
	}
	if roundOrder.Table != firstOrder.Table || roundOrder.Status != "OPEN" {
		t.Fatalf("expected the new tile to carry the same table and be OPEN, got %+v", roundOrder)
	}
	var roundItems []kdsmodels.KDSItems
	if err := db.Where("kds_order_id = ?", roundOrder.ID).Find(&roundItems).Error; err != nil {
		t.Fatal(err)
	}
	if len(roundItems) != 1 || roundItems[0].Name != "pizza" {
		t.Fatalf("expected the new item to land on the new round's own tile, got %+v", roundItems)
	}
}

func TestProcessMessage_VoidOnlyTicketStillMergesIntoExistingOrder(t *testing.T) {
	_, dbPath := setupPollerTestDB(t)
	DBLocation = dbPath
	FileLocation = t.TempDir()

	obj := kdsmodels.KDSPollObject{}
	obj.Config = kdsmodels.KDSConfig{}
	ProcessMessage(obj, []byte(d01Round1Ticket), FileLocation, dbPath)

	// A void-only ticket for the same reference carries no new items - it
	// must still find and mark the existing burger, not get swept into the
	// new-round-gets-its-own-tile path (which is scoped to tickets that
	// actually carry new items).
	voidTicket := " * dine in *\n" +
		"table 9\n" +
		"* * kitchen * *\n" +
		"ipad3/sam\n" +
		"10:22 pm a1.1\n" +
		"voided item\n" +
		"1 x burger\n" +
		"\x1dvb\x00\n"
	ProcessMessage(obj, []byte(voidTicket), FileLocation, dbPath)

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	defer closeConnection(db)

	var orders []kdsmodels.KDSOrders
	if err := db.Where("reference LIKE ?", "a1.1%").Find(&orders).Error; err != nil {
		t.Fatal(err)
	}
	if len(orders) != 1 {
		t.Fatalf("a void-only ticket must never spin up its own new tile, got %d order(s): %+v", len(orders), orders)
	}
}
