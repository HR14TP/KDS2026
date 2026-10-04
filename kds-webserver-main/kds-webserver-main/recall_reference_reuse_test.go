package main

import (
	"testing"

	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// D08: getDatabaseOrderFromReferenceAnyStatus drops the status=OPEN filter
// (the whole point is finding a COMPLETE order to reopen), so a reference
// reused later the same day can match more than one row. table is the one
// piece of context a recall carries that can actually tell those rows
// apart - see the function's own doc comment. Reuses
// setupCloseOrderTestDB/getTimeSQL from closeorder_scope_test.go - same
// package, same isolated in-memory DB pattern.

func mustCreateOrderAtTable(t *testing.T, db *gorm.DB, reference string, table string) kdsmodels.KDSOrders {
	t.Helper()
	order := kdsmodels.KDSOrders{Reference: reference, Table: table, Status: "OPEN", Created: getTimeSQL()}
	require.NoError(t, db.Create(&order).Error)
	return order
}

func TestGetDatabaseOrderFromReferenceAnyStatus_PrefersTableMatchOverRecency(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	// Same reference reused later the same day for a genuinely different
	// table's order - the older row (table 5) is the one this recall
	// actually means, even though a newer row (table 12) also matches the
	// bare reference.
	older := mustCreateOrderAtTable(t, db, "a1.2", "table 5")
	mustCreateOrderAtTable(t, db, "a1.2", "table 12")

	got := getDatabaseOrderFromReferenceAnyStatus("a1.2", "table 5")
	assert.Equal(t, older.ID, got.ID, "must resolve to the candidate whose table matches, not just the most recent one")
}

func TestGetDatabaseOrderFromReferenceAnyStatus_FallsBackToRecencyWhenNoTableMatch(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	mustCreateOrderAtTable(t, db, "a1.2", "table 5")
	newest := mustCreateOrderAtTable(t, db, "a1.2", "table 12")

	got := getDatabaseOrderFromReferenceAnyStatus("a1.2", "table 99")
	assert.Equal(t, newest.ID, got.ID, "no candidate matches the given table - must fail open to the most recent row, not find nothing")
}

func TestGetDatabaseOrderFromReferenceAnyStatus_BlankTableFallsBackToRecency(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	mustCreateOrderAtTable(t, db, "a1.2", "table 5")
	newest := mustCreateOrderAtTable(t, db, "a1.2", "table 12")

	// An older peer that hasn't been upgraded yet sends no Table at all -
	// must behave exactly as before this fix.
	got := getDatabaseOrderFromReferenceAnyStatus("a1.2", "")
	assert.Equal(t, newest.ID, got.ID)
}

func TestGetDatabaseOrderFromReferenceAnyStatus_TableMatchIsCaseInsensitive(t *testing.T) {
	db := setupCloseOrderTestDB(t)
	older := mustCreateOrderAtTable(t, db, "a1.2", "Table 5")
	mustCreateOrderAtTable(t, db, "a1.2", "table 12")

	got := getDatabaseOrderFromReferenceAnyStatus("a1.2", "TABLE 5")
	assert.Equal(t, older.ID, got.ID)
}

func TestGetDatabaseOrderFromReferenceAnyStatus_NoMatchingReferenceReturnsZeroValue(t *testing.T) {
	setupCloseOrderTestDB(t)
	got := getDatabaseOrderFromReferenceAnyStatus("no-such-reference", "table 5")
	assert.Equal(t, 0, got.ID)
}

func TestGetDatabaseOrderFromReferenceAnyStatus_SingleCandidateIgnoresTableMismatch(t *testing.T) {
	// Only one order with this reference exists at all - no ambiguity to
	// resolve, so an unrelated/stale table value on the recall message
	// must not cause a miss.
	db := setupCloseOrderTestDB(t)
	only := mustCreateOrderAtTable(t, db, "a1.3", "table 7")

	got := getDatabaseOrderFromReferenceAnyStatus("a1.3", "table 40")
	assert.Equal(t, only.ID, got.ID)
}
