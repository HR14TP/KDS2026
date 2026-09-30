package contactlessactions

import (
	"fmt"
	"net/url"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/reverent911/kds-golang-lib/pkg/contactless/contactlessmodels"
	"github.com/reverent911/kds-golang-lib/pkg/generic/errors"
	"github.com/reverent911/kds-golang-lib/pkg/generic/helper"
	"github.com/reverent911/kds-golang-lib/pkg/ikentoo/ikentooactions"
	"github.com/reverent911/kds-golang-lib/pkg/ikentoo/ikentoomodels"
)

var Debug = true
var Globict = make(map[int]string, 100)

// ShowDebug ... ShowDebug
func ShowDebug(i interface{}) {
	if Debug {
		fmt.Println(i)
	}
}
func HandleErrors(err error) {
	if err != nil {
		errors.CaptureSentryError(errors.GetSentryServices().CONTACTLESS_SYNC, err)
	}
}

// UpdateMenuImport ... Update Menu Import
func UpdateMenuImport(theurl string) {
	token := GetContactlessIDSFromWooCommerce(theurl)
	menu := ikentooactions.GetMenuFromID(token.Token, token.MenuID, token.BID)
	for i := range menu.MenuEntryGroups {
		UpdateProcessEntry(theurl, menu.MenuEntryGroups[i])
	}
}

// UpdateProcessEntry ... Process Main Entry
func UpdateProcessEntry(theurl string, m ikentoomodels.MenuEntryGroup) {
	if m.Type == "group" {
		for i := range m.MenuEntries {
			if m.MenuEntries[i].Type == "group" {
				UpdateProcessCategory(theurl, m.MenuEntries[i])
			} else if m.MenuEntries[i].Type == "menuItem" {
				UpdateProcessSimpleItem(theurl, m.MenuEntries[i].SKU, m.MenuEntries[i].ProductName, m.MenuEntries[i].ProductPrice)
			} else if m.MenuEntries[i].Type == "menuDeal" {
				UpdateProcessMenuDeal(theurl, m.MenuEntries[i])
			}
		}
	}
}

// UpdateProcessCategory ... Update Contactless Category
func UpdateProcessCategory(theurl string, m ikentoomodels.MenuEntry) {
	for i := range m.MenuEntries {
		if m.MenuEntries[i].Type == "group" {
			UpdateProcessCategory(theurl, m.MenuEntries[i])
		}
		if m.MenuEntries[i].Type == "menuItem" {
			UpdateProcessSimpleItem(theurl, m.MenuEntries[i].SKU, m.MenuEntries[i].ProductName, m.MenuEntries[i].ProductPrice)
		}
		if m.MenuEntries[i].Type == "menuDeal" {
			UpdateProcessMenuDeal(theurl, m.MenuEntries[i])
		}
	}
}

// UpdateProcessSimpleItem ... Update Simple Item
func UpdateProcessSimpleItem(theurl string, sku string, name string, price string) contactlessmodels.ContactlessSimpleItem {
	item := MenuUpdateSimpleItem(theurl, sku, name, price)
	return item
}

// UpdateProcessMenuDeal ... Update Menu Deal
func UpdateProcessMenuDeal(theurl string, m ikentoomodels.MenuEntry) {
	//menuDealGroups
	var atties []int
	var atties2 []int
	for i := range m.MenuDealGroups {
		attr := MenuCreateAttribute(theurl, url.QueryEscape(m.MenuDealGroups[i].Description))
		if attr.ID > 0 {
			for j := range m.MenuDealGroups[i].MenuDealGroupItems {
				MenuCreateAttributeTerm(theurl, m.MenuDealGroups[i].MenuDealGroupItems[j].ProductName, m.MenuDealGroups[i].MenuDealGroupItems[j].ProductPrice, helper.IntToString(attr.ID), m.MenuDealGroups[i].MenuDealGroupItems[j].SKU)
			}
		}
		if m.MenuDealGroups[i].MultiSelectionPermitted {
			// Create Checkboxes
			ca := MenuCreateCustomAttribute(theurl, helper.IntToString(attr.ID), "checkbox", m.MenuDealGroups[i].MustSelectAnItem)
			atties2 = append(atties2, ca.ID)
		} else if len(m.MenuDealGroups[i].MenuDealGroupItems) > 0 {
			// Create Selects
			ca := MenuCreateCustomAttribute(theurl, helper.IntToString(attr.ID), "select", m.MenuDealGroups[i].MustSelectAnItem)
			atties2 = append(atties2, ca.ID)
		} else {
			//atties = append(atties, attr.ID)
		}
	}
	var theprod contactlessmodels.ContactlessSimpleItem
	if len(atties) > 0 {
		//Variable Product
		// theprod = ProcessVariableItem(theurl, m.SKU, m.ProductName, m.ProductPrice, atties, baseid)
	}
	if len(atties2) > 0 {
		if theprod.ID > 0 {
			//assign
			MenuAssignCustomMeta(theurl, theprod.ID, atties2)
		} else {
			theprod = UpdateProcessSimpleItem(theurl, m.SKU, m.ProductName, m.ProductPrice)
			if theprod.ID > 0 {
				MenuAssignCustomMeta(theurl, theprod.ID, atties2)
			}
		}
	}
	if !(theprod.ID > 0) {
		theprod = UpdateProcessSimpleItem(theurl, m.SKU, m.ProductName, m.ProductPrice)
	}
}

// FullMenuImport ... Full Menu Import
func FullMenuImport(theurl string) {
	token := GetContactlessIDSFromWooCommerce(theurl)
	menu := ikentooactions.GetMenuFromID(token.Token, token.MenuID, token.BID)
	for i := range menu.MenuEntryGroups {
		ProcessEntry(theurl, menu.MenuEntryGroups[i], helper.StringToInt(token.BaseID))
	}
}

// NewFullMenuImport ... Full Menu Import
func NewFullMenuImport(theurl string) {
	token := GetContactlessIDSFromWooCommerce(theurl)
	menu := ikentooactions.GetMenuFromID(token.Token, token.MenuID, token.BID)
	for i := range menu.MenuEntryGroups {
		ProcessNewFullEntry(theurl, menu.MenuEntryGroups[i], helper.StringToInt(token.BaseID))
	}
}

// ProcessEntry ... Process Main Entry
func ProcessEntry(theurl string, m ikentoomodels.MenuEntryGroup, baseid int) {
	if m.Type == "group" {
		//Category Needs To Be Made
		category := MenuImportCategory(theurl, url.QueryEscape(m.Name), baseid)
		if category.ID > 0 {
			// Category Created
			for i := range m.MenuEntries {
				if m.MenuEntries[i].Type == "group" {
					ProcessCategory(theurl, m.MenuEntries[i], category.ID)
				} else if m.MenuEntries[i].Type == "menuItem" {
					ProcessSimpleItem(theurl, m.MenuEntries[i].SKU, m.MenuEntries[i].ProductName, m.MenuEntries[i].ProductPrice, category.ID)
				} else if m.MenuEntries[i].Type == "menuDeal" {
					ProcessMenuDeal(theurl, m.MenuEntries[i], category.ID)
				}
			}
		}
	}
}

// ProcessCategory ... Import Contactless Category
func ProcessCategory(theurl string, m ikentoomodels.MenuEntry, baseid int) {
	category := MenuImportCategory(theurl, url.QueryEscape(m.Name), baseid)
	time.Sleep(1 * time.Second)
	for i := range m.MenuEntries {
		if m.MenuEntries[i].Type == "group" {
			ProcessCategory(theurl, m.MenuEntries[i], category.ID)
		}
		if m.MenuEntries[i].Type == "menuItem" {
			ProcessSimpleItem(theurl, m.MenuEntries[i].SKU, m.MenuEntries[i].ProductName, m.MenuEntries[i].ProductPrice, category.ID)
		}
		if m.MenuEntries[i].Type == "menuDeal" {
			ProcessMenuDeal(theurl, m.MenuEntries[i], category.ID)
		}
	}
}

// ProcessSimpleItem ... Create Simple Item
func ProcessSimpleItem(theurl string, sku string, name string, price string, baseid int) contactlessmodels.ContactlessSimpleItem {
	item := MenuCreateSimpleItem(theurl, sku, name, price, baseid)
	Globict[baseid] += sku + ","
	spew.Dump(Globict)
	time.Sleep(1 * time.Second)
	return item
}

// ProcessVariableItem ... Create Variable Item
func ProcessVariableItem(theurl string, sku string, name string, price string, attrs []int, baseid int) contactlessmodels.ContactlessSimpleItem {
	item := MenuCreateVariableItem(theurl, sku, name, price, attrs, baseid)
	Globict[baseid] += sku + ","
	spew.Dump(Globict)
	time.Sleep(1 * time.Second)
	return item
}

// ProcessMenuDeal ... Process Menu Deal
func ProcessMenuDeal(theurl string, m ikentoomodels.MenuEntry, baseid int) {
	//menuDealGroups
	var atties []int
	var atties2 []int
	for i := range m.MenuDealGroups {
		attr := MenuCreateAttribute(theurl, url.QueryEscape(m.MenuDealGroups[i].Description))
		if attr.ID > 0 {
			for j := range m.MenuDealGroups[i].MenuDealGroupItems {
				MenuCreateAttributeTerm(theurl, m.MenuDealGroups[i].MenuDealGroupItems[j].ProductName, m.MenuDealGroups[i].MenuDealGroupItems[j].ProductPrice, helper.IntToString(attr.ID), m.MenuDealGroups[i].MenuDealGroupItems[j].SKU)
				time.Sleep(1 * time.Second)
			}
		}
		if m.MenuDealGroups[i].MultiSelectionPermitted {
			// Create Checkboxes
			ca := MenuCreateCustomAttribute(theurl, helper.IntToString(attr.ID), "checkbox", m.MenuDealGroups[i].MustSelectAnItem)
			atties2 = append(atties2, ca.ID)
		} else if len(m.MenuDealGroups[i].MenuDealGroupItems) > 0 {
			// Create Selects
			ca := MenuCreateCustomAttribute(theurl, helper.IntToString(attr.ID), "select", m.MenuDealGroups[i].MustSelectAnItem)
			atties2 = append(atties2, ca.ID)
		} else {
			//atties = append(atties, attr.ID)
		}
		time.Sleep(1 * time.Second)
	}
	var theprod contactlessmodels.ContactlessSimpleItem
	if len(atties) > 0 {
		//Variable Product
		theprod = ProcessVariableItem(theurl, m.SKU, m.ProductName, m.ProductPrice, atties, baseid)
	}
	if len(atties2) > 0 {
		if theprod.ID > 0 {
			//assign
			MenuAssignCustomMeta(theurl, theprod.ID, atties2)
		} else {
			theprod = ProcessSimpleItem(theurl, m.SKU, m.ProductName, m.ProductPrice, baseid)
			if theprod.ID > 0 {
				MenuAssignCustomMeta(theurl, theprod.ID, atties2)
			}
		}
	}
	if !(theprod.ID > 0) {
		theprod = ProcessSimpleItem(theurl, m.SKU, m.ProductName, m.ProductPrice, baseid)
	}
	time.Sleep(1 * time.Second)
}

// RichItemImport ... Rich Item Import
func RichItemImport(theurl string, businessid string) {
	token := GetContactlessIDSFromWooCommerce(theurl)
	teller := 1
	items := ikentooactions.GetRichItems(token.Token, businessid, teller)
	for len(items.Embedded.RichItemDtoList) > 0 {
		//spew.Dump(len(items.Embedded.RichItemDtoList))
		for i := range items.Embedded.RichItemDtoList {
			//spew.Dump(len(items.Embedded.RichItemDtoList[i].Descriptions))
			for j := range items.Embedded.RichItemDtoList[i].Descriptions {
				if len(items.Embedded.RichItemDtoList[i].Descriptions[j].Description) > 1 {
					fmt.Println("Updating : " + items.Embedded.RichItemDtoList[i].SKU + " : " + items.Embedded.RichItemDtoList[i].Descriptions[j].Description)
					MenuUpdateDescriptionUpdate(theurl, items.Embedded.RichItemDtoList[i].SKU, items.Embedded.RichItemDtoList[i].Descriptions[j].Description)
				}
			}
		}
		teller++
		fmt.Println("Next Page")
		items = ikentooactions.GetRichItems(token.Token, businessid, teller)
	}
}
