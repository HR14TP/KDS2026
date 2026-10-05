package main

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupKnownTerminalsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&kdsmodels.KDSKnownTerminals{}))
	ENV = &kdsmodels.KDSEnv{DB: db}
	return db
}

func TestUpsertKnownTerminal_CreatesNewRow(t *testing.T) {
	db := setupKnownTerminalsTestDB(t)
	upsertKnownTerminal("pnp", "172.24.31.62")

	all := []kdsmodels.KDSKnownTerminals{}
	require.NoError(t, db.Find(&all).Error)
	require.Len(t, all, 1)
	assert.Equal(t, "pnp", all[0].Screen)
	assert.Equal(t, "172.24.31.62", all[0].IP)
}

// A terminal's IP can legitimately change (DHCP lease renewal, a swapped
// NIC, etc) - re-announcing must update the existing row rather than
// accumulating a second one for the same screen name.
func TestUpsertKnownTerminal_UpdatesIPOnReannounceRatherThanDuplicating(t *testing.T) {
	db := setupKnownTerminalsTestDB(t)
	upsertKnownTerminal("pnp", "172.24.31.62")
	upsertKnownTerminal("pnp", "172.24.31.99")

	all := []kdsmodels.KDSKnownTerminals{}
	require.NoError(t, db.Find(&all).Error)
	require.Len(t, all, 1, "re-announcing the same screen must update, not duplicate")
	assert.Equal(t, "172.24.31.99", all[0].IP)
}

func TestUpsertKnownTerminal_ScreenMatchIsCaseInsensitive(t *testing.T) {
	db := setupKnownTerminalsTestDB(t)
	upsertKnownTerminal("PNP", "172.24.31.62")
	upsertKnownTerminal("pnp", "172.24.31.99")

	all := []kdsmodels.KDSKnownTerminals{}
	require.NoError(t, db.Find(&all).Error)
	require.Len(t, all, 1)
	assert.Equal(t, "172.24.31.99", all[0].IP)
}

func TestUpsertKnownTerminal_BlankScreenOrIPIsNoop(t *testing.T) {
	db := setupKnownTerminalsTestDB(t)
	upsertKnownTerminal("", "172.24.31.62")
	upsertKnownTerminal("pnp", "")

	all := []kdsmodels.KDSKnownTerminals{}
	require.NoError(t, db.Find(&all).Error)
	assert.Empty(t, all)
}

func TestListKnownTerminals_ExcludesStaleEntries(t *testing.T) {
	db := setupKnownTerminalsTestDB(t)
	fresh := kdsmodels.KDSKnownTerminals{Screen: "pnp", IP: "172.24.31.62", LastSeen: time.Now().Format("2006-01-02 15:04:05")}
	stale := kdsmodels.KDSKnownTerminals{Screen: "bakery", IP: "172.24.31.242", LastSeen: time.Now().Add(-10 * time.Minute).Format("2006-01-02 15:04:05")}
	require.NoError(t, db.Create(&fresh).Error)
	require.NoError(t, db.Create(&stale).Error)

	result := listKnownTerminals()

	require.Len(t, result, 1)
	assert.Equal(t, "pnp", result[0].Screen)
}

// A LastSeen this terminal can't parse is kept rather than hidden, matching
// listAvailablePassTerminals' own behavior - a formatting mismatch must
// never silently erase a real terminal from the list.
func TestListKnownTerminals_UnparseableLastSeenIsKeptNotHidden(t *testing.T) {
	db := setupKnownTerminalsTestDB(t)
	row := kdsmodels.KDSKnownTerminals{Screen: "pnp", IP: "172.24.31.62", LastSeen: "not-a-timestamp"}
	require.NoError(t, db.Create(&row).Error)

	result := listKnownTerminals()

	require.Len(t, result, 1)
	assert.Equal(t, "pnp", result[0].Screen)
}
