package ylaactions

import "github.com/reverent911/kds-golang-lib/pkg/generic/database"

func YLAGetFullItemDetail(SKU string, ikentooID int) {
	db := database.YLAConnection()
	item := YLAItemFindBySKUDB(SKU, ikentooID, db)
	item.Sequences = YLAItemGetSequencesForItem(item, db)
	item.ProductionInstructions = YLAItemGetModifiersForItem(item, db)
}
