package contactlessactions

import (
	"net/url"

	"github.com/reverent911/kds-golang-lib/pkg/contactless/contactlessmodels"
	"github.com/reverent911/kds-golang-lib/pkg/generic/helper"
	"github.com/reverent911/kds-golang-lib/pkg/ikentoo/ikentoomodels"
)

// ProcessEntry ... Process Main Entry
func ProcessNewFullEntry(theurl string, m ikentoomodels.MenuEntryGroup, baseid int) {
	if m.Type == "group" {
		//Category Needs To Be Made
		category := MenuImportCategory(theurl, url.QueryEscape(m.Name), baseid)
		if category.ID > 0 {
			// Category Created
			for i := range m.MenuEntries {
				if m.MenuEntries[i].Type == "group" {
					ProcessNewFullCategory(theurl, m.MenuEntries[i], category.ID)
				} else if m.MenuEntries[i].Type == "menuItem" {
					ProcessNewFullSimpleItem(theurl, m.MenuEntries[i].SKU, m.MenuEntries[i].ProductName, m.MenuEntries[i].ProductPrice, category.ID)
				} else if m.MenuEntries[i].Type == "menuDeal" {
					ProcessNewFullMenuDeal(theurl, m.MenuEntries[i], category.ID)
				}
			}
		}
	}
}

// ProcessCategory ... Import Contactless Category
func ProcessNewFullCategory(theurl string, m ikentoomodels.MenuEntry, baseid int) {
	category := MenuImportCategory(theurl, url.QueryEscape(m.Name), baseid)
	for i := range m.MenuEntries {
		if m.MenuEntries[i].Type == "group" {
			ProcessNewFullCategory(theurl, m.MenuEntries[i], category.ID)
		}
		if m.MenuEntries[i].Type == "menuItem" {
			ProcessNewFullSimpleItem(theurl, m.MenuEntries[i].SKU, m.MenuEntries[i].ProductName, m.MenuEntries[i].ProductPrice, category.ID)
		}
		if m.MenuEntries[i].Type == "menuDeal" {
			ProcessNewFullMenuDeal(theurl, m.MenuEntries[i], category.ID)
		}
	}
}

// ProcessFullSimpleItem ... Create Simple Item
func ProcessNewFullSimpleItem(theurl string, sku string, name string, price string, baseid int) contactlessmodels.ContactlessSimpleItem {
	item := MenuCreateSimpleItem(theurl, sku, name, price, baseid)
	Globict[baseid] += sku + ","
	return item
}

// ProcessFullMenuDeal ... Process Menu Deal
func ProcessNewFullMenuDeal(theurl string, m ikentoomodels.MenuEntry, baseid int) {
	ShowDebug("ProcessNewFullMenuDeal")
	var atties []int
	var atties2 []int
	for i := range m.MenuDealGroups {
		termname := m.SKU + "-" + m.MenuDealGroups[i].Description
		attr := WebsiteCreateAttribute(theurl, url.QueryEscape(termname))
		if attr.ID > 0 {
			for j := range m.MenuDealGroups[i].MenuDealGroupItems {
				WebsiteCreateAttributeTerm(theurl, m.MenuDealGroups[i].MenuDealGroupItems[j].ProductName, m.MenuDealGroups[i].MenuDealGroupItems[j].ProductPrice, helper.IntToString(attr.ID), m.MenuDealGroups[i].MenuDealGroupItems[j].SKU)
			}
		}
		if m.MenuDealGroups[i].MultiSelectionPermitted {
			// Create Checkboxes
			ca := WebsiteCreateCustomAttributeCombinedWithNameAndSKU(theurl, helper.IntToString(attr.ID), "checkbox", m.MenuDealGroups[i].MustSelectAnItem, url.QueryEscape(m.MenuDealGroups[i].Description), m.SKU)
			atties2 = append(atties2, ca.ID)
		} else if len(m.MenuDealGroups[i].MenuDealGroupItems) > 0 {
			// Create Selects
			ca := WebsiteCreateCustomAttributeCombinedWithNameAndSKU(theurl, helper.IntToString(attr.ID), "select", m.MenuDealGroups[i].MustSelectAnItem, url.QueryEscape(m.MenuDealGroups[i].Description), m.SKU)
			atties2 = append(atties2, ca.ID)
		}
	}
	var theprod contactlessmodels.ContactlessSimpleItem
	if len(atties) > 0 {
		//Variable Product
		theprod = ProcessNewFullVariableItem(theurl, m.SKU, m.ProductName, m.ProductPrice, atties, baseid)
	}
	if len(atties2) > 0 {
		if theprod.ID > 0 {
			//assign
			WebsiteAssignCustomMetaSKU(theurl, m.SKU, atties2)
		} else {
			theprod = ProcessFullSimpleItem(theurl, m.SKU, m.ProductName, m.ProductPrice, baseid)
			if theprod.ID > 0 {
				WebsiteAssignCustomMetaSKU(theurl, m.SKU, atties2)
			}
		}
	}
	if !(theprod.ID > 0) {
		theprod = ProcessFullSimpleItem(theurl, m.SKU, m.ProductName, m.ProductPrice, baseid)
	}
	//}
}

// ProcessFullMenuDeal ... Process Menu Deal
func ProcessNewFullMenuDealPriceOnly(theurl string, m ikentoomodels.MenuEntry, baseid int) {
	ShowDebug("ProcessNewFullMenuDeal")
	ProcessFullSimpleItem(theurl, m.SKU, m.ProductName, m.ProductPrice, baseid)
}

// ProcessFullSimpleItem ... Create Simple Item
func ProcessFullSimpleItem(theurl string, sku string, name string, price string, baseid int) contactlessmodels.ContactlessSimpleItem {
	item := MenuCreateSimpleItem(theurl, sku, name, price, baseid)
	Globict[baseid] += sku + ","
	//spew.Dump(globdict)
	//time.Sleep(1 * time.Second)
	return item
}
