package contactlessactions

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/reverent911/kds-golang-lib/pkg/generic/helper"
	"github.com/reverent911/kds-golang-lib/pkg/ikentoo/ikentooactions"
	"github.com/reverent911/kds-golang-lib/pkg/ikentoo/ikentoomodels"
)

func SingleItemMenuImport(theurl string, sku string) {
	token := GetContactlessIDSFromWooCommerce(theurl)
	//spew.Dump(token)
	menu := ikentooactions.GetMenuFromID(token.Token, token.MenuID, token.BID)
	//spew.Dump(menu)
	for i := range menu.MenuEntryGroups {
		ProcessNewFullEntrySingle(theurl, menu.MenuEntryGroups[i], helper.StringToInt(token.BaseID), sku)
	}
}

func SingleItemMenuImportPriceOnly(theurl string, sku string) {
	token := GetContactlessIDSFromWooCommerce(theurl)
	//spew.Dump(token)
	menu := ikentooactions.GetMenuFromID(token.Token, token.MenuID, token.BID)
	//spew.Dump(menu)
	for i := range menu.MenuEntryGroups {
		ProcessNewFullEntrySinglePriceOnly(theurl, menu.MenuEntryGroups[i], helper.StringToInt(token.BaseID), sku)
	}
}

// ProcessEntry ... Process Main Entry
func ProcessNewFullEntrySingle(theurl string, m ikentoomodels.MenuEntryGroup, baseid int, sku string) {
	if m.Type == "group" {
		for i := range m.MenuEntries {
			//spew.Dump(m.MenuEntries[i].SKU)
			//spew.Dump(sku)
			if m.MenuEntries[i].Type == "group" {
				ProcessNewFullCategorySingle(theurl, m.MenuEntries[i], 0, sku)
			} else if m.MenuEntries[i].Type == "menuItem" && m.MenuEntries[i].SKU == sku {
				spew.Dump("IMPORTING ITEM NOW 4")
				ProcessNewFullSimpleItem(theurl, m.MenuEntries[i].SKU, m.MenuEntries[i].ProductName, m.MenuEntries[i].ProductPrice, 0)
			} else if m.MenuEntries[i].Type == "menuDeal" && m.MenuEntries[i].SKU == sku {
				spew.Dump("IMPORTING ITEM NOW 5")
				ProcessNewFullMenuDeal(theurl, m.MenuEntries[i], 0)
			}
		}
	}
}

// ProcessEntry ... Process Main Entry Price Only
func ProcessNewFullEntrySinglePriceOnly(theurl string, m ikentoomodels.MenuEntryGroup, baseid int, sku string) {
	if m.Type == "group" {
		for i := range m.MenuEntries {
			//spew.Dump(m.MenuEntries[i].SKU)
			//spew.Dump(sku)
			if m.MenuEntries[i].Type == "group" {
				ProcessNewFullCategorySinglePriceOnly(theurl, m.MenuEntries[i], 0, sku)
			} else if m.MenuEntries[i].Type == "menuItem" && m.MenuEntries[i].SKU == sku {
				spew.Dump("IMPORTING ITEM NOW 4")
				ProcessNewFullSimpleItem(theurl, m.MenuEntries[i].SKU, m.MenuEntries[i].ProductName, m.MenuEntries[i].ProductPrice, 0)
			} else if m.MenuEntries[i].Type == "menuDeal" && m.MenuEntries[i].SKU == sku {
				spew.Dump("IMPORTING ITEM NOW 5")
				ProcessNewFullMenuDealPriceOnly(theurl, m.MenuEntries[i], 0)
			}
		}
	}
}

// ProcessCategory ... Import Contactless Category
func ProcessNewFullCategorySingle(theurl string, m ikentoomodels.MenuEntry, baseid int, sku string) {
	for i := range m.MenuEntries {
		if m.MenuEntries[i].Type == "group" {
			ProcessNewFullCategorySingle(theurl, m.MenuEntries[i], 0, sku)
		}
		if m.MenuEntries[i].Type == "menuItem" && m.MenuEntries[i].SKU == sku {
			spew.Dump("IMPORTING ITEM NOW 3")
			ProcessNewFullSimpleItem(theurl, m.MenuEntries[i].SKU, m.MenuEntries[i].ProductName, m.MenuEntries[i].ProductPrice, 0)
		}
		if m.MenuEntries[i].Type == "menuDeal" && m.MenuEntries[i].SKU == sku {
			spew.Dump("IMPORTING ITEM NOW 2")
			ProcessNewFullMenuDeal(theurl, m.MenuEntries[i], 0)
		}
	}
}

// ProcessNewFullCategorySinglePriceOnly ... Import Contactless Category Price Only
func ProcessNewFullCategorySinglePriceOnly(theurl string, m ikentoomodels.MenuEntry, baseid int, sku string) {
	for i := range m.MenuEntries {
		if m.MenuEntries[i].Type == "group" {
			ProcessNewFullCategorySinglePriceOnly(theurl, m.MenuEntries[i], 0, sku)
		}
		if m.MenuEntries[i].Type == "menuItem" && m.MenuEntries[i].SKU == sku {
			spew.Dump("IMPORTING ITEM NOW 3")
			ProcessNewFullSimpleItem(theurl, m.MenuEntries[i].SKU, m.MenuEntries[i].ProductName, m.MenuEntries[i].ProductPrice, 0)
		}
		if m.MenuEntries[i].Type == "menuDeal" && m.MenuEntries[i].SKU == sku {
			spew.Dump("IMPORTING ITEM NOW 2")
			ProcessNewFullMenuDealPriceOnly(theurl, m.MenuEntries[i], 0)
		}
	}
}
