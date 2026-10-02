package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
	"gorm.io/gorm"
)

// D07: confirmed live against real venue ticket samples - a stray leading
// 0x1D ("group separator") control byte on a build-your-own dish's deeper
// option lines (e.g. "Breakfast Your Way" -> "One Egg, Your Way" -> a
// further-indented side choice) threw off GetItemsMatching's byte-offset
// indentation checks, wrongly promoting the real side (e.g. "Crispy Hash
// Brown") to its own standalone main item instead of a sub-item of the
// dish, and causing it to wrongly absorb the dish's other, genuinely
// indented sides (bacon, toast, egg white) as if they were its own.
//
// This reproduces the exact byte shape found on two independent real
// tickets from this venue (table 14 and table 58, both "Breakfast Your
// Way" orders): a normal 2-space-indented sub-item line, then a 0x1D byte
// followed by 5 spaces and a sub-sub-item with no "x" (folds onto the
// prior sub's name via GetProductionInstructions - unaffected by this
// fix), then a 0x1D byte followed by 4 spaces and a genuine "1 x <name>"
// side dish line (the one this fix targets), then two more ordinary
// 2-space-indented sides.
const realVenueTicketWithControlByte = " * dine in *\n" +
	"table 14\n" +
	"* * kitchen * *\n" +
	"ipad3/percy\n" +
	"10:19 pm a1.1\n" +
	"1 x breakfast your way\n" +
	"  1 x one egg, your way\n" +
	"\x1d     1 hard fried\n" +
	"\x1d    1 x crispy hash brown\n" +
	"  1 x bacon (60g)\n" +
	"  1 x toasted ciabatta\n" +
	"  1 x egg white\n" +
	"\x1dvb\x00\n"

func setupPollerTestDB(t *testing.T) (*gorm.DB, string) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "kds.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	schema, err := os.ReadFile("testdata/schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(string(schema)).Error; err != nil {
		t.Fatal(err)
	}
	closeConnection(db)
	return db, dbPath
}

func TestProcessMessage_ControlByteDoesNotSplitBuildYourOwnDish(t *testing.T) {
	_, dbPath := setupPollerTestDB(t)
	DBLocation = dbPath
	FileLocation = t.TempDir()

	obj := kdsmodels.KDSPollObject{}
	obj.Config = kdsmodels.KDSConfig{}
	ProcessMessage(obj, []byte(realVenueTicketWithControlByte), FileLocation, dbPath)

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	defer closeConnection(db)

	var items []kdsmodels.KDSItems
	if err := db.Find(&items).Error; err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected exactly one main item (the whole dish must not split into two stray tiles), got %d: %+v", len(items), items)
	}
	if items[0].Name != "breakfast your way" {
		t.Fatalf("expected the single item to be 'breakfast your way', got %q", items[0].Name)
	}

	var subs []kdsmodels.KDSSubitems
	if err := db.Where("kds_item_id = ?", items[0].ID).Find(&subs).Error; err != nil {
		t.Fatal(err)
	}
	gotNames := map[string]bool{}
	for _, s := range subs {
		gotNames[s.Name] = true
	}
	wantNames := []string{
		"one egg, your way - 1 hard fried",
		"crispy hash brown",
		"bacon (60g)",
		"toasted ciabatta",
		"egg white",
	}
	if len(subs) != len(wantNames) {
		t.Fatalf("expected %d sub-items on the one dish, got %d: %+v", len(wantNames), len(subs), subs)
	}
	for _, want := range wantNames {
		if !gotNames[want] {
			t.Errorf("missing expected sub-item %q on 'breakfast your way' - got %+v", want, gotNames)
		}
	}
}
