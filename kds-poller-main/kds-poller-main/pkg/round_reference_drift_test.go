package main

import (
	"github.com/glebarez/sqlite"
	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
	"gorm.io/gorm"
	"testing"
)

// Each physical ticket is already scoped to exactly one screen upstream (see
// screenReg's single "if !obj.Foundscreen" latch in engine.go - a ticket
// only ever carries one "* * <screen> * *" section) - a round that has
// nothing for a given station simply never produces a ticket for that
// station's own poller at all. Every terminal runs its own poller
// independently against its own local DB with no cross-machine
// coordination, which used to let two terminals land on different
// round-tile Reference suffixes for the exact same round the moment a
// prior round had skipped one of them (a per-terminal SubOrders counter,
// incremented only on rounds that touched that terminal's own screen) -
// breaking the by-Reference match a remote close/recall relies on. Fixed
// by deriving the suffix from the ticket's own printed time instead (see
// engine.go's round-tile splitting branch) - identical on every physical
// copy of the same round's ticket regardless of which screen reads it.

const kitchenRound1 = " * dine in *\n" +
	"table 9\n" +
	"* * kitchen * *\n" +
	"ipad3/sam\n" +
	"10:19 pm a1.1\n" +
	"1 x burger\n" +
	"\x1dvb\x00\n"

const kitchenRound2 = " * dine in *\n" +
	"table 9\n" +
	"* * kitchen * *\n" +
	"ipad3/sam\n" +
	"10:25 pm a1.1\n" +
	"1 x fries\n" +
	"\x1dvb\x00\n"

const kitchenRound3 = " * dine in *\n" +
	"table 9\n" +
	"* * kitchen * *\n" +
	"ipad3/sam\n" +
	"10:40 pm a1.1\n" +
	"1 x steak\n" +
	"\x1dvb\x00\n"

const bakeryRound1 = " * dine in *\n" +
	"table 9\n" +
	"* * bakery * *\n" +
	"ipad3/sam\n" +
	"10:19 pm a1.1\n" +
	"1 x bread roll\n" +
	"\x1dvb\x00\n"

// bakeryRound2 deliberately doesn't exist - round 2 (fries) has nothing for
// bakery, so bakery's own poller never even receives a ticket for it.

const bakeryRound3 = " * dine in *\n" +
	"table 9\n" +
	"* * bakery * *\n" +
	"ipad3/sam\n" +
	"10:40 pm a1.1\n" +
	"1 x cake\n" +
	"\x1dvb\x00\n"

// TestProcessMessage_RoundSuffixStaysConsistentWhenARoundSkipsAScreen
// guards against a live symptom: closing an order on the pass terminal
// ("headchef"/kitchen) didn't close it on a linked sub-screen (e.g.
// "bakery"), because round 2 (fries) had nothing for bakery, and bakery's
// own per-terminal SubOrders counter fell behind kitchen's - so by round 3
// (which touches both again) each terminal computed a DIFFERENT "-R%d"
// suffix for what should be the same round-tile, and a remote close naming
// one terminal's own suffix found nothing on the other's
// (getDatabaseOrderFromReference - "Invalid Order", a silent no-op). Fixed
// by deriving the suffix from the ticket's own printed time instead of the
// per-terminal counter - both terminals must compute the exact same
// Reference for round 3 even though bakery never saw round 2 at all.
func TestProcessMessage_RoundSuffixStaysConsistentWhenARoundSkipsAScreen(t *testing.T) {
	_, kitchenDBPath := setupPollerTestDB(t)
	_, bakeryDBPath := setupPollerTestDB(t)
	obj := kdsmodels.KDSPollObject{}
	obj.Config = kdsmodels.KDSConfig{}

	FileLocation = t.TempDir()
	DBLocation = kitchenDBPath
	ProcessMessage(obj, []byte(kitchenRound1), FileLocation, kitchenDBPath)
	ProcessMessage(obj, []byte(kitchenRound2), FileLocation, kitchenDBPath)
	ProcessMessage(obj, []byte(kitchenRound3), FileLocation, kitchenDBPath)

	DBLocation = bakeryDBPath
	ProcessMessage(obj, []byte(bakeryRound1), FileLocation, bakeryDBPath)
	// Round 2 (fries) never reaches bakery - nothing for it to do.
	ProcessMessage(obj, []byte(bakeryRound3), FileLocation, bakeryDBPath)

	kitchenDB, err := gorm.Open(sqlite.Open(kitchenDBPath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	defer closeConnection(kitchenDB)
	bakeryDB, err := gorm.Open(sqlite.Open(bakeryDBPath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	defer closeConnection(bakeryDB)

	var kitchenOrders, bakeryOrders []kdsmodels.KDSOrders
	if err := kitchenDB.Where("reference LIKE ?", "a1.1%").Order("id asc").Find(&kitchenOrders).Error; err != nil {
		t.Fatal(err)
	}
	if err := bakeryDB.Where("reference LIKE ?", "a1.1%").Order("id asc").Find(&bakeryOrders).Error; err != nil {
		t.Fatal(err)
	}

	// Kitchen saw all 3 rounds touch it: original + 2 round-tiles.
	if len(kitchenOrders) != 3 {
		t.Fatalf("expected kitchen to have 3 orders (original + 2 rounds), got %d: %+v", len(kitchenOrders), kitchenOrders)
	}
	// Bakery only saw rounds 1 and 3 touch it: original + 1 round-tile.
	if len(bakeryOrders) != 2 {
		t.Fatalf("expected bakery to have 2 orders (original + 1 round), got %d: %+v", len(bakeryOrders), bakeryOrders)
	}

	kitchenRound3Tile := kitchenOrders[len(kitchenOrders)-1]
	bakeryRound3Tile := bakeryOrders[len(bakeryOrders)-1]

	t.Logf("kitchen round-3 tile reference: %q", kitchenRound3Tile.Reference)
	t.Logf("bakery round-3 tile reference: %q", bakeryRound3Tile.Reference)

	// Both tiles represent the SAME round 3 (same table, same ticket
	// timestamp "10:40 pm") - despite bakery never having seen round 2 at
	// all (so its own SubOrders counter sits one behind kitchen's), both
	// terminals must still compute the exact same Reference, because the
	// suffix comes from the ticket's own printed time rather than either
	// terminal's own private counter.
	if kitchenRound3Tile.Reference != bakeryRound3Tile.Reference {
		t.Fatalf("expected both terminals to agree on round 3's reference despite bakery skipping round 2, got kitchen=%q bakery=%q", kitchenRound3Tile.Reference, bakeryRound3Tile.Reference)
	}
	if kitchenRound3Tile.Reference != "a1.1-R1040pm" {
		t.Fatalf("expected the round-3 reference derived from the ticket's own printed time, got %q", kitchenRound3Tile.Reference)
	}
}
