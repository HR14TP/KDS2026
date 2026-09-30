package ylaactions

import (
	"math"

	"github.com/reverent911/kds-golang-lib/pkg/generic/helper"
	"github.com/reverent911/kds-golang-lib/pkg/yla/ylamodels"
	"gorm.io/gorm"
)

// YLAMenuFromBaseAloneLite ... Menu From Base Alone
func YLAMenuFromBaseAloneLiteLite(id int, clientid int, db *gorm.DB) ylamodels.YLAMenu {
	data := ylamodels.YLAMenu{}
	db.Where("id = ?", id).First(&data)
	if data.ID > 0 {
		db.Where("parent_screen_id = ?", data.ID).Where("deleted_at IS NULL").Order("order_no asc").Find(&data.Menus)
		//subQuery := db.Select("item_id").Where("menu_id = ?", data.ID).Where("ikentoo_business_id = ?", clientid).Table("items").Table("menu_items")
		//db.Where("sku IN (?)", subQuery).Where("ikentoo_business_id = ?", clientid).Find(&data.Items)
		/*for i := range data.Menus {
			db.Where("parent_screen_id = ?", data.Menus[i].ID).Order("order_no asc").Find(&data.Menus[i].Menus)
			subQuery1 := db.Select("item_id").Where("menu_id = ?", data.Menus[i].ID).Where("ikentoo_business_id = ?", clientid).Table("items").Table("menu_items")
			db.Where("sku IN (?)", subQuery1).Where("ikentoo_business_id = ?", clientid).Find(&data.Menus[i].Items)
		}*/
	}
	return data
}

// YLAUserMenuFromBaseLite ... User Menu From Base Lite
func YLAUserMenuFromBaseLiteNEW(id int, clientid int, userid int, db *gorm.DB) (int, interface{}) {
	/* need to check if user is part of a membership */
	var hasmembership = false
	usr := ylamodels.YLAUser{}
	ms := ylamodels.YLAMembership{}

	if userid > 0 {
		db.Where("id = ?", userid).First(&usr)
		if usr.ID > 0 && usr.MembershipID > 0 {
			/* user is a member of a group */
			db.Where("id = ?", usr.MembershipID).First(&ms)
			if ms.ID > 0 && ms.Type != "PRICE" {
				hasmembership = true
			}
		}
	}

	data := ylamodels.YLAMenu{}
	db.Where("id = ?", id).First(&data)
	if data.ID > 0 {
		db.Where("parent_screen_id = ?", data.ID).Where("deleted_at IS NULL").Order("order_no asc").Find(&data.Menus)
		/*V2*/
		subQuery := db.Select("item_id").Where("menu_id = ?", data.ID).Where("ikentoo_business_id = ?", clientid).Table("menu_items")
		db.Where("sku IN (?)", subQuery).Where("ikentoo_business_id = ?", clientid).Where("disabled = ?", 0).Find(&data.Items)
		for i := range data.Items {
			if hasmembership {
				discount := helper.StringToFloat(ms.Type)
				price := data.Items[i].CostPrice
				val := price - ((price * discount) / 100)
				data.Items[i].CostPrice = math.Round(val)
			}
			if data.Items[i].PhotoID != "" {
				db.Where("id = ?", data.Items[i].PhotoID).Find(&data.Items[i].File)
			}

			/*V2*/
			subQuery1 := db.Select("sequence_id").Where("item_sku = ?", data.Items[i].Sku).Where("ikentoo_business_id = ?", clientid).Table("item_sequences")
			db.Where("id IN (?)", subQuery1).Where("ikentoo_business_id = ?", clientid).Where("deleted_at is NULL").Order("`order` asc").Find(&data.Items[i].Sequences)
			for j := range data.Items[i].Sequences {
				db.Where("sequence_id = ?", data.Items[i].Sequences[j].ID).Where("deleted_at IS NULL").Find(&data.Items[i].Sequences[j].SequenceValues)
			}

			/*V2*/
			subQuery2 := db.Select("production_instruction_id").Where("item_sku = ?", data.Items[i].Sku).Where("ikentoo_business_id = ?", clientid).Table("item_production_instructions")
			db.Where("id IN (?)", subQuery2).Where("ikentoo_business_id = ?", clientid).Where("deleted_at is NULL").Find(&data.Items[i].ProductionInstructions)
			for k := range data.Items[i].ProductionInstructions {
				db.Where("production_instruction_id = ?", data.Items[i].ProductionInstructions[k].ID).Where("deleted_at IS NULL").Find(&data.Items[i].ProductionInstructions[k].ProductionInstructionValue)
			}
		}
	}
	return 200, data
}

// YLANewMenuFromBaseAloneLiteLite ... Menu From Base Alone
func YLANewMenuFromBaseAloneLiteLite(id int, clientid int, db *gorm.DB) ylamodels.YLAMenu {
	data := ylamodels.YLAMenu{}
	db.Where("id = ?", id).First(&data)
	if data.ID > 0 {
		db.Where("parent_screen_id = ?", data.ID).Where("deleted_at IS NULL").Order("order_no asc").Find(&data.Menus)
	}
	return data
}
