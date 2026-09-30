package main

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupCloseOrderTestDB gives each test its own isolated in-memory sqlite DB
// with a real ENV.DB, matching how these functions run in production
// (package-level ENV.DB, not dependency-injected) - tests must not run
// t.Parallel() against each other since they all mutate the same package
// global.
func setupCloseOrderTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&kdsmodels.KDSOrders{}, &kdsmodels.KDSItems{}, &kdsmodels.KDSSubitems{}, &kdsmodels.KDSScreens{}))
	ENV = &kdsmodels.KDSEnv{DB: db}
	return db
}

func mustCreateScreen(t *testing.T, db *gorm.DB, name string, assignedPassTerminal string) kdsmodels.KDSScreens {
	t.Helper()
	screen := kdsmodels.KDSScreens{Name: name, AssignedPassTerminal: assignedPassTerminal}
	require.NoError(t, db.Create(&screen).Error)
	return screen
}

func mustCreateOrder(t *testing.T, db *gorm.DB, reference string) kdsmodels.KDSOrders {
	t.Helper()
	order := kdsmodels.KDSOrders{Reference: reference, Status: "OPEN", Created: getTimeSQL()}
	require.NoError(t, db.Create(&order).Error)
	return order
}

func mustCreateItem(t *testing.T, db *gorm.DB, orderID int, screenID int, name string, status string) kdsmodels.KDSItems {
	t.Helper()
	item := kdsmodels.KDSItems{KDSOrderID: orderID, KDSScreenID: screenID, Name: name, Qty: 1, Status: status, Created: getTimeSQL()}
	require.NoError(t, db.Create(&item).Error)
	return item
}

func assertOrderStatus(t *testing.T, db *gorm.DB, orderID int, want string) {
	t.Helper()
	var order kdsmodels.KDSOrders
	require.NoError(t, db.Where("id = ?", orderID).First(&order).Error)
	assert.Equal(t, want, order.Status, "order status")
}

// --- closeOrderScreenDatabase ---

// The original bug this whole mechanism fixes: a REMOTE close (allowFinalize
// false) scoped to a screen that IS relevant to this row must still never
// finalize the row just because everything on it happens to already be
// complete - only a genuinely local press, or a genuinely deferred pass
// terminal, may finalize.
func TestCloseOrderScreenDatabase_RemoteCloseNeverFinalizesEvenWhenRelevantAndFullyComplete(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	bar := mustCreateScreen(t, db, "bar", "")
	kitchen := mustCreateScreen(t, db, "kitchen", "")
	order := mustCreateOrder(t, db, "a1214093.1353")
	mustCreateItem(t, db, order.ID, bar.ID, "sprite", "COMPLETE")
	mustCreateItem(t, db, order.ID, kitchen.ID, "steak", "COMPLETE")

	// Simulates applyCloseOrderScreenRemote receiving bar's own close,
	// replayed on kitchen's local copy - relevant (matches the bar item on
	// this row) but not deferred (kitchen isn't assigned to bar), so
	// allowFinalize is false.
	closeOrderScreenDatabase(order.ID, "bar", false)
	assertOrderStatus(t, db, order.ID, "OPEN")
}

func TestCloseOrderScreenDatabase_LocalCloseFinalizesWhenEverythingComplete(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	kitchen := mustCreateScreen(t, db, "kitchen", "")
	order := mustCreateOrder(t, db, "ref-1")
	mustCreateItem(t, db, order.ID, kitchen.ID, "steak", "OPEN")

	closeOrderScreenDatabase(order.ID, "kitchen", true)
	assertOrderStatus(t, db, order.ID, "COMPLETE")
}

func TestCloseOrderScreenDatabase_LocalCloseDoesNotFinalizeWhileOtherScreenStillOpen(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	kitchen := mustCreateScreen(t, db, "kitchen", "")
	bar := mustCreateScreen(t, db, "bar", "")
	order := mustCreateOrder(t, db, "ref-2")
	mustCreateItem(t, db, order.ID, kitchen.ID, "steak", "OPEN")
	mustCreateItem(t, db, order.ID, bar.ID, "sprite", "OPEN")

	closeOrderScreenDatabase(order.ID, "kitchen", true)
	assertOrderStatus(t, db, order.ID, "OPEN")

	var barItem kdsmodels.KDSItems
	require.NoError(t, db.Where("kds_screen_id = ?", bar.ID).First(&barItem).Error)
	assert.Equal(t, "OPEN", barItem.Status, "kitchen's close must never touch bar's own item")
}

// A remote close from a screen this row genuinely has nothing to do with
// (no matching screen item, no orphan) must never finalize, even with
// allowFinalize true - relevant gates independently of allowFinalize.
func TestCloseOrderScreenDatabase_IrrelevantScreenNeverFinalizesEvenWithAllowFinalizeTrue(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	kitchen := mustCreateScreen(t, db, "kitchen", "")
	mustCreateScreen(t, db, "milkshakes", "")
	order := mustCreateOrder(t, db, "ref-3")
	mustCreateItem(t, db, order.ID, kitchen.ID, "steak", "COMPLETE")

	closeOrderScreenDatabase(order.ID, "milkshakes", true)
	assertOrderStatus(t, db, order.ID, "OPEN")
}

// A genuinely deferred pass terminal's remote close (allowFinalize true,
// computed by isDeferredToPassTerminal) legitimately finalizes. A
// pass-terminal-assigned screen's close always arrives with Passmode=true
// (see applyCloseOrderScreenRemote's dispatch), so this replays through
// closeOrderForPassScreen, not closeOrderScreenDatabase - it's the
// assigned_pass_terminal match, not a plain name match, that finds pizza's
// item here.
func TestCloseOrderForPassScreen_DeferredPassTerminalRemoteCloseFinalizes(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	pizza := mustCreateScreen(t, db, "pizza", "headchef")
	order := mustCreateOrder(t, db, "ref-4")
	mustCreateItem(t, db, order.ID, pizza.ID, "margherita", "COMPLETE")

	require.True(t, isDeferredToPassTerminal("headchef"))
	closeOrderForPassScreen(order.ID, "headchef", isDeferredToPassTerminal("headchef"))
	assertOrderStatus(t, db, order.ID, "COMPLETE")
}

// Blank-named-screen orphan sweep counts as relevant on its own.
func TestCloseOrderScreenDatabase_OrphanItemAloneCountsAsRelevantAndFinalizes(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	order := mustCreateOrder(t, db, "ref-5")
	mustCreateItem(t, db, order.ID, 0, "unparsed-item", "COMPLETE") // kds_screen_id 0 -> orphan

	closeOrderScreenDatabase(order.ID, "some-screen-nobody-has", true)
	assertOrderStatus(t, db, order.ID, "COMPLETE")
}

// The gating must not be sticky: once a remote close is correctly blocked,
// a later genuinely-local close on the same row must still finalize once
// everything really is done.
func TestCloseOrderScreenDatabase_SubsequentLocalCloseStillFinalizesAfterBlockedRemoteClose(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	bar := mustCreateScreen(t, db, "bar", "")
	kitchen := mustCreateScreen(t, db, "kitchen", "")
	order := mustCreateOrder(t, db, "ref-6")
	mustCreateItem(t, db, order.ID, bar.ID, "sprite", "COMPLETE")
	mustCreateItem(t, db, order.ID, kitchen.ID, "steak", "OPEN")

	// Bar's remote close reaches kitchen's row first - blocked, as above.
	closeOrderScreenDatabase(order.ID, "bar", false)
	assertOrderStatus(t, db, order.ID, "OPEN")

	// Kitchen now presses its own local Close Order.
	closeOrderScreenDatabase(order.ID, "kitchen", true)
	assertOrderStatus(t, db, order.ID, "COMPLETE")
}

func TestCloseOrderScreenDatabase_OrderNotFoundIsNoop(t *testing.T) {
	setupCloseOrderTestDB(t)
	result := closeOrderScreenDatabase(999999, "kitchen", true)
	assert.Equal(t, 0, result.ID)
}

func TestCloseOrderScreenDatabase_ScreenNameMatchIsCaseInsensitive(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	kitchen := mustCreateScreen(t, db, "Kitchen", "")
	order := mustCreateOrder(t, db, "ref-7")
	mustCreateItem(t, db, order.ID, kitchen.ID, "steak", "OPEN")

	closeOrderScreenDatabase(order.ID, "KITCHEN", true)
	assertOrderStatus(t, db, order.ID, "COMPLETE")
}

// --- closeOrderForPassScreen ---

// The K07 headline scenario: a pass screen's force-close must complete its
// own+assigned sections but never touch, and never finalize on behalf of, a
// genuinely separate terminal's item that happens to share this row.
func TestCloseOrderForPassScreen_DoesNotFinalizeOnUnrelatedSeparateTerminalItem(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	headchef := mustCreateScreen(t, db, "headchef", "")
	bar := mustCreateScreen(t, db, "bar", "")
	order := mustCreateOrder(t, db, "ref-8")
	mustCreateItem(t, db, order.ID, headchef.ID, "steak", "OPEN")
	mustCreateItem(t, db, order.ID, bar.ID, "sprite", "COMPLETE")

	// headchef force-bumps its own section locally.
	closeOrderForPassScreen(order.ID, "headchef", true)

	var barItem kdsmodels.KDSItems
	require.NoError(t, db.Where("kds_screen_id = ?", bar.ID).First(&barItem).Error)
	assert.Equal(t, "COMPLETE", barItem.Status, "bar's own item was already complete independently, untouched by this close")
	assertOrderStatus(t, db, order.ID, "COMPLETE")
}

func TestCloseOrderForPassScreen_ForceClosesAssignedSectionRegardlessOfItemStatus(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	headchef := mustCreateScreen(t, db, "headchef", "")
	pizza := mustCreateScreen(t, db, "pizza", "headchef")
	order := mustCreateOrder(t, db, "ref-9")
	mustCreateItem(t, db, order.ID, headchef.ID, "steak", "OPEN")
	mustCreateItem(t, db, order.ID, pizza.ID, "margherita", "OPEN")

	closeOrderForPassScreen(order.ID, "headchef", true)

	var pizzaItem kdsmodels.KDSItems
	require.NoError(t, db.Where("kds_screen_id = ?", pizza.ID).First(&pizzaItem).Error)
	assert.Equal(t, "COMPLETE", pizzaItem.Status, "headchef's bump force-closes its assigned pizza section too")
	assertOrderStatus(t, db, order.ID, "COMPLETE")
}

// A remote pass-screen close that ISN'T genuinely relevant here (this row
// has no item on that screen or any of its assigned sections, no orphan)
// must not finalize even with allowFinalize true.
func TestCloseOrderForPassScreen_IrrelevantRemoteCloseNeverFinalizes(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	kitchen := mustCreateScreen(t, db, "kitchen", "")
	mustCreateScreen(t, db, "headchef", "")
	order := mustCreateOrder(t, db, "ref-10")
	mustCreateItem(t, db, order.ID, kitchen.ID, "steak", "COMPLETE")

	closeOrderForPassScreen(order.ID, "headchef", true)
	assertOrderStatus(t, db, order.ID, "OPEN")
}

// The exact bug found after final37 shipped: a remote close IS relevant
// (this row carries a duplicate of the remote screen's own item) but the
// remote screen is not genuinely deferred-to by this terminal - must not
// finalize purely because everything on the row happens to be complete.
func TestCloseOrderForPassScreen_RelevantButNotDeferredRemoteCloseNeverFinalizes(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	bar := mustCreateScreen(t, db, "bar", "")
	kitchen := mustCreateScreen(t, db, "kitchen", "")
	order := mustCreateOrder(t, db, "ref-11")
	mustCreateItem(t, db, order.ID, bar.ID, "sprite", "COMPLETE")
	mustCreateItem(t, db, order.ID, kitchen.ID, "prawns", "COMPLETE")

	require.False(t, isDeferredToPassTerminal("bar"))
	closeOrderForPassScreen(order.ID, "bar", isDeferredToPassTerminal("bar"))
	assertOrderStatus(t, db, order.ID, "OPEN")
}

func TestCloseOrderForPassScreen_OrphanItemAloneCountsAsRelevantAndFinalizes(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	order := mustCreateOrder(t, db, "ref-12")
	mustCreateItem(t, db, order.ID, 0, "unparsed-item", "OPEN")

	closeOrderForPassScreen(order.ID, "headchef", true)
	assertOrderStatus(t, db, order.ID, "COMPLETE")
}

// --- isDeferredToPassTerminal ---

func TestIsDeferredToPassTerminal(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	mustCreateScreen(t, db, "pizza", "headchef")
	mustCreateScreen(t, db, "bar", "")

	assert.True(t, isDeferredToPassTerminal("headchef"))
	assert.True(t, isDeferredToPassTerminal("HEADCHEF"), "case-insensitive")
	assert.False(t, isDeferredToPassTerminal("bar"))
	assert.False(t, isDeferredToPassTerminal("nobody-assigned-here"))
}

// --- orderHasItemsMatching ---

func TestOrderHasItemsMatching(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	kitchen := mustCreateScreen(t, db, "kitchen", "")
	order := mustCreateOrder(t, db, "ref-13")
	mustCreateItem(t, db, order.ID, kitchen.ID, "steak", "OPEN")

	assert.True(t, orderHasItemsMatching(order.ID, "kds_screen_id = ?", kitchen.ID))
	assert.False(t, orderHasItemsMatching(order.ID, "kds_screen_id = ?", 999999))
	assert.False(t, orderHasItemsMatching(999999, "kds_screen_id = ?", kitchen.ID), "nonexistent order")
}
