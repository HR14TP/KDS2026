package ylaactions

import (
	"github.com/reverent911/kds-golang-lib/pkg/generic/database"
	"github.com/reverent911/kds-golang-lib/pkg/yla/ylamodels"
)

func GetMobileSuperClientLite(clientid int) (int, interface{}) {
	var db = database.YLAConnection()
	data := ylamodels.YLACompany{}
	db.Where("ikentoo_id = ?", clientid).First(&data)
	db.Where("ikentoo_b_id = ?", clientid).Find(&data.Pages)
	if data.ID > 0 {
		//db.Model(&data).Association("Branches").Find(&data.Branches)
		db.Where("company_id = ?", data.ID).Where("deleted_at IS NULL").Find(&data.Branches)
		for i := range data.Branches {
			data.Branches[i].Menu = YLAMenuFromBaseAloneLiteLite(data.Branches[i].MenuID, clientid, db)
			//db.Model(&data.Branches[i]).Association("Hours").Find(&data.Branches[i].Hours)
			db.Where("branch_id = ?", &data.Branches[i].ID).Find(&data.Branches[i].Hours)
			db.Where("branch_id = ?", &data.Branches[i].ID).Find(&data.Branches[i].BranchPaymentType)
			db.Where("ikentoo_business_id = ?", clientid).Where("sku = ?", &data.Branches[i].DeliverySKU).Find(&data.Branches[i].DeliveryItem)
			db.Where("ikentoo_business_id = ?", clientid).Where("sku = ?", &data.Branches[i].TipSKU).Find(&data.Branches[i].TipItem)
		}
	}
	return 200, data
}

// YLANewGetClientConfig ... get Client Config From ID
func YLANewGetClientConfig(clientID int) ylamodels.YLAClientConfig {
	var clientconfig ylamodels.YLAClientConfig
	var db = database.YLAConnection()
	db.Where("client_id = ?", clientID).First(&clientconfig)
	return clientconfig
}

func YLANewGetClientLight(clientid int) (int, interface{}) {
	var db = database.YLAConnection()
	data := ylamodels.YLACompany{}
	db.Where("ikentoo_id = ?", clientid).First(&data)
	db.Where("ikentoo_b_id = ?", clientid).Find(&data.Pages)
	if data.ID > 0 {
		db.Where("company_id = ?", data.ID).Where("deleted_at IS NULL").Find(&data.Branches)
		for i := range data.Branches {
			data.Branches[i].Menu = YLANewMenuFromBaseAloneLiteLite(data.Branches[i].MenuID, clientid, db)
			db.Where("branch_id = ?", &data.Branches[i].ID).Find(&data.Branches[i].Hours)
			db.Where("branch_id = ?", &data.Branches[i].ID).Find(&data.Branches[i].BranchPaymentType)
			db.Where("ikentoo_business_id = ?", clientid).Where("sku = ?", &data.Branches[i].DeliverySKU).Find(&data.Branches[i].DeliveryItem)
			db.Where("ikentoo_business_id = ?", clientid).Where("sku = ?", &data.Branches[i].TipSKU).Find(&data.Branches[i].TipItem)
		}
	}
	return 200, data
}
