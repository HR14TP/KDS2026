package contactlessactions

import (
	"github.com/reverent911/kds-golang-lib/pkg/contactless/contactlessmodels"
	"github.com/reverent911/kds-golang-lib/pkg/generic/database"
	"gorm.io/gorm"
)

// GetContactlessSite ... Get contactless site
func GetContactlessSite(db *gorm.DB, locationid string) contactlessmodels.Site {
	site := contactlessmodels.Site{}
	db.Where("location_id = ?", locationid).First(&site)
	return site
}

func StartPizzaBabyStaging(sku string) {
	siteUrl := "https://staging.pizzababy.co.za/"
	SingleItemMenuImport(siteUrl, sku)
}

func StartPizzaBabyLive(sku string) {
	siteUrl := "https://www.pizzababy.co.za/"
	SingleItemMenuImport(siteUrl, sku)
}

func StartPizzaBabyLivePriceMainOnly(sku string) {
	siteUrl := "https://www.pizzababy.co.za/"
	SingleItemMenuImportPriceOnly(siteUrl, sku)
}

func StartPizzaBabyTest(sku string) {
	siteUrl := "https://pizza-baby.co.za/"
	SingleItemMenuImport(siteUrl, sku)
}

func StartPizzaBabyTestPriceMainOnly(sku string) {
	siteUrl := "https://pizza-baby.co.za/"
	SingleItemMenuImportPriceOnly(siteUrl, sku)
}

// https://staging.pizzababy.co.za/
func GetContactlessJobs() {
	db := database.ContactlessConnection()
	inst := []contactlessmodels.Instruction{}
	db.Where("status = ?", 0).Order("created_at asc").Find(&inst)
	for i := range inst {
		site := GetContactlessSite(db, inst[i].LocationID)
		if inst[i].Instruction == "FULL" {
			WebsiteUnlinkProducts(site.SiteURL)
			WebStartMaintenance(site.SiteURL)
			NewFullMenuImport(site.SiteURL)
			//doItemChecks(site.SiteURL)
			RichItemImport(site.SiteURL, site.IkentooID)
			WebEndMaintenance(site.SiteURL)
			inst[i].Status = 1
			db.Save(inst[i])
		} else if inst[i].Instruction == "DESC" {
			//WebStartMaintenance(site.SiteURL)
			RichItemImport(site.SiteURL, site.IkentooID)
			//WebEndMaintenance(site.SiteURL)
			inst[i].Status = 1
			db.Save(inst[i])
		} else if inst[i].Instruction == "UPDATE" {
			WebStartMaintenance(site.SiteURL)
			NewFullMenuImport(site.SiteURL)
			RichItemImport(site.SiteURL, site.IkentooID)
			WebEndMaintenance(site.SiteURL)
			inst[i].Status = 1
			db.Save(inst[i])
		}
		WebEndMaintenance(site.SiteURL)
		UpdateContactlessJobStatus(site.SiteURL, inst[i].Instruction, "DONE")
	}
}
