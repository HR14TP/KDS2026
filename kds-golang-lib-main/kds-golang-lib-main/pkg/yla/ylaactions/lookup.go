package ylaactions

import (
	"github.com/reverent911/kds-golang-lib/pkg/generic/helper"
	"github.com/reverent911/kds-golang-lib/pkg/yla/ylamodels"
	"gorm.io/gorm"
)

func YLAGetUserByEmailAndPassword(email string, password string, db *gorm.DB) ylamodels.YLAAdminUser {
	ylaaccount := YLAGetAccountFromEmail(email, db)
	ylauser := ylamodels.YLAAdminUser{}
	if ylaaccount.ID > 0 {
		ylausers := []ylamodels.YLAAdminUser{}
		db.Where("email = ?", email).Find(&ylausers)
		for i := range ylausers {
			if helper.AuthCheckPassword(ylausers[i].Password, password) {
				//this is an admin user and correct password , now check the ikentoo IDs
				if ylaaccount.IkentooID == helper.StringToInt(ylausers[i].IkentooBusinessID) {
					//everything is good
					ylauser = ylausers[i]
				}
			}
		}
	}
	return ylauser
}

func YLAGetAccountFromEmail(email string, db *gorm.DB) ylamodels.YLAAccount {
	ylaaccount := ylamodels.YLAAccount{}
	db.Where("admin_email = ?", email).First(&ylaaccount)
	return ylaaccount
}

// ---------------- ITEMS ------------------ //
func YLAGetItemsFromIkentooID(ikentooid int, db *gorm.DB) []ylamodels.YLAItem {
	items := []ylamodels.YLAItem{}
	db.Where("ikentoo_business_id = ?", ikentooid).Where("deleted_at IS NULL").Find(&items)
	return items
}

// ItemFindBySKUDB ... Item Find By SKU
func YLAItemFindBySKUDB(sku string, ikentooID int, db *gorm.DB) ylamodels.YLAItem {
	item := ylamodels.YLAItem{}
	db.Where("sku = ?", sku).Where("ikentoo_business_id = ?", ikentooID).First(&item)
	return item
}

func YLAItemGetSequencesForItem(item ylamodels.YLAItem, db *gorm.DB) []ylamodels.YLASequence {
	ikentooid := helper.StringToInt(item.IkentooBusinessID)
	subQuery1 := db.Select("sequence_id").Where("item_sku = ?", item.Sku).Where("ikentoo_business_id = ?", ikentooid).Table("item_sequences")
	db.Where("id IN (?)", subQuery1).Where("ikentoo_business_id = ?", ikentooid).Order("`order` asc").Find(&item.Sequences)
	for j := range item.Sequences {
		db.Where("sequence_id = ?", item.Sequences[j].ID).Find(&item.Sequences[j].SequenceValues)
	}
	return item.Sequences
}

func YLAItemGetModifiersForItem(item ylamodels.YLAItem, db *gorm.DB) []ylamodels.YLAProductionInstruction {
	ikentooid := helper.StringToInt(item.IkentooBusinessID)
	subQuery1 := db.Select("production_instruction_id").Where("item_sku = ?", item.Sku).Where("ikentoo_business_id = ?", ikentooid).Table("item_production_instructions")
	db.Where("id IN (?)", subQuery1).Where("ikentoo_business_id = ?", ikentooid).Order("`order` asc").Find(&item.ProductionInstructions)
	for j := range item.ProductionInstructions {
		db.Where("production_instrcution_id = ?", item.ProductionInstructions[j].ID).Find(&item.ProductionInstructions[j].ProductionInstructionValue)
	}
	return item.ProductionInstructions
}

// ---------------- END ITEMS ------------------ //
