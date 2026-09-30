package ylaactions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/reverent911/kds-golang-lib/pkg/generic/database"
	"github.com/reverent911/kds-golang-lib/pkg/generic/helper"
	"github.com/reverent911/kds-golang-lib/pkg/yla/ylamodels"

	"github.com/davecgh/go-spew/spew"
)

func YLAGetFullOrderFromID(orderid int) ylamodels.YLAOrder {
	var db = database.YLAConnection()
	var ordertotal = float64(0)
	order := ylamodels.YLAOrder{}
	db.Where("id = ?", orderid).First(&order)
	db.Model(&order).Association("OrderItems").Find(&order.OrderItems)
	db.Where("ikentoo_id = ?", order.BranchID).Find(&order.Branch)
	db.Where("id = ?", order.UserID).Find(&order.User)
	for i := range order.OrderItems {
		db.Where("sku = ?", order.OrderItems[i].ItemSku).Where("ikentoo_business_id = ?", order.Branch.IkentooBusinessID).Find(&order.OrderItems[i].Item)
		ordertotal += float64(float64(order.OrderItems[i].Qty) * order.OrderItems[i].Amount)
		db.Where("item_sku = ?", order.OrderItems[i].ItemSku).Where("order_id = ?", order.OrderItems[i].OrderID).Where("order_item_id = ?", order.OrderItems[i].ItemID).Find(&order.OrderItems[i].OrderItemSequences)
		db.Where("item_sku = ?", order.OrderItems[i].ItemSku).Where("order_id = ?", order.OrderItems[i].OrderID).Find(&order.OrderItems[i].OrderItemProductionInstructions)
		for j := range order.OrderItems[i].OrderItemSequences {
			db.Where("id = ?", order.OrderItems[i].OrderItemSequences[j].SequenceValueID).Find(&order.OrderItems[i].OrderItemSequences[j].SequenceValues)
			for k := range order.OrderItems[i].OrderItemSequences[j].SequenceValues {
				ordertotal += float64(order.OrderItems[i].OrderItemSequences[j].SequenceValues[k].Price)
			}
		}
		for j := range order.OrderItems[i].OrderItemProductionInstructions {
			db.Model(&order.OrderItems[i].OrderItemProductionInstructions[j]).Association("ProductionValues").Find(&order.OrderItems[i].OrderItemProductionInstructions[j].ProductionValues)
		}
	}
	order.OrderTotal = ordertotal
	return order
}

func YLAGetFullOrderFromReference(reference string) ylamodels.YLAOrder {
	var db = database.YLAConnection()
	var ordertotal = float64(0)
	order := ylamodels.YLAOrder{}
	db.Where("external_reference = ?", reference).First(&order)
	db.Model(&order).Association("OrderItems").Find(&order.OrderItems)
	db.Where("ikentoo_id = ?", order.BranchID).Find(&order.Branch)
	db.Where("id = ?", order.UserID).Find(&order.User)
	for i := range order.OrderItems {
		db.Where("sku = ?", order.OrderItems[i].ItemSku).Where("ikentoo_business_id = ?", order.Branch.IkentooBusinessID).Find(&order.OrderItems[i].Item)
		ordertotal += float64(float64(order.OrderItems[i].Qty) * order.OrderItems[i].Amount)
		db.Where("item_sku = ?", order.OrderItems[i].ItemSku).Where("order_id = ?", order.OrderItems[i].OrderID).Where("order_item_id = ?", order.OrderItems[i].ItemID).Find(&order.OrderItems[i].OrderItemSequences)
		db.Where("item_sku = ?", order.OrderItems[i].ItemSku).Where("order_id = ?", order.OrderItems[i].OrderID).Find(&order.OrderItems[i].OrderItemProductionInstructions)
		for j := range order.OrderItems[i].OrderItemSequences {
			db.Where("id = ?", order.OrderItems[i].OrderItemSequences[j].SequenceValueID).Find(&order.OrderItems[i].OrderItemSequences[j].SequenceValues)
			for k := range order.OrderItems[i].OrderItemSequences[j].SequenceValues {
				ordertotal += float64(order.OrderItems[i].OrderItemSequences[j].SequenceValues[k].Price)
			}
		}
		for j := range order.OrderItems[i].OrderItemProductionInstructions {
			db.Model(&order.OrderItems[i].OrderItemProductionInstructions[j]).Association("ProductionValues").Find(&order.OrderItems[i].OrderItemProductionInstructions[j].ProductionValues)
		}
	}
	order.OrderTotal = ordertotal
	return order
}

func YLAGetFullOrderFromCollectionCodeAndBranch(collectioncode string, branchid int64) ylamodels.YLAOrder {
	var db = database.YLAConnection()
	var ordertotal = float64(0)
	order := ylamodels.YLAOrder{}
	db.Where("ikentoo_collection_code = ?", collectioncode).Where("branch_id = ?", branchid).First(&order)
	db.Model(&order).Association("OrderItems").Find(&order.OrderItems)
	db.Where("ikentoo_id = ?", order.BranchID).Find(&order.Branch)
	db.Where("id = ?", order.UserID).Find(&order.User)
	for i := range order.OrderItems {
		db.Where("sku = ?", order.OrderItems[i].ItemSku).Where("ikentoo_business_id = ?", order.Branch.IkentooBusinessID).Find(&order.OrderItems[i].Item)
		ordertotal += float64(float64(order.OrderItems[i].Qty) * order.OrderItems[i].Amount)
		db.Where("item_sku = ?", order.OrderItems[i].ItemSku).Where("order_id = ?", order.OrderItems[i].OrderID).Where("order_item_id = ?", order.OrderItems[i].ItemID).Find(&order.OrderItems[i].OrderItemSequences)
		db.Where("item_sku = ?", order.OrderItems[i].ItemSku).Where("order_id = ?", order.OrderItems[i].OrderID).Find(&order.OrderItems[i].OrderItemProductionInstructions)
		for j := range order.OrderItems[i].OrderItemSequences {
			db.Where("id = ?", order.OrderItems[i].OrderItemSequences[j].SequenceValueID).Find(&order.OrderItems[i].OrderItemSequences[j].SequenceValues)
			for k := range order.OrderItems[i].OrderItemSequences[j].SequenceValues {
				ordertotal += float64(order.OrderItems[i].OrderItemSequences[j].SequenceValues[k].Price)
			}
		}
		for j := range order.OrderItems[i].OrderItemProductionInstructions {
			db.Model(&order.OrderItems[i].OrderItemProductionInstructions[j]).Association("ProductionValues").Find(&order.OrderItems[i].OrderItemProductionInstructions[j].ProductionValues)
		}
	}
	order.OrderTotal = ordertotal
	return order
}

func GetOrderDetailsFromID(orderid int) ylamodels.YLAOrderDetails {
	var db = database.YLAConnection()
	details := ylamodels.YLAOrderDetails{}
	db.Where("order_id = ?", orderid).First(&details)
	return details
}
func GetAccountFromIkentooID(id int) ylamodels.YLAAccount {
	var db = database.YLAConnection()
	acc := ylamodels.YLAAccount{}
	db.Where("ikentoo_id = ?", id).Find(&acc)
	return acc
}
func YLAOrderItemToIkentooOrderItem(oitem ylamodels.YLAOrderItem) ylamodels.OrderItemIkentoo {
	tempItem := ylamodels.OrderItemIkentoo{}
	tempItem.ItemSku = oitem.ItemSku
	tempItem.Quantity = oitem.Qty
	tempItem.ItemPrice = oitem.Amount
	tempItem.SubItems = YLAOrderItemSequenceToIkentooOrderItemSubitem(oitem.OrderItemSequences)
	tempItem.SubItems = append(tempItem.SubItems, YLAOrderItemProductionToIkentooOrderItemSubitem(oitem.OrderItemProductionInstructions)...)
	return tempItem
}
func YLAOrderItemSequenceToIkentooOrderItemSubitem(oitemsequences []ylamodels.YLAOrderItemSequence) []ylamodels.SubItems {
	subitems := []ylamodels.SubItems{}
	for j := range oitemsequences {
		tempSubItem := ylamodels.SubItems{}
		tempSubItem.ItemSku = helper.IntToString(oitemsequences[j].SequenceValues[0].Sku)
		tempSubItem.ItemPrice = oitemsequences[j].SequenceValues[0].Price
		tempSubItem.ItemName = oitemsequences[j].SequenceValues[0].Name
		subitems = append(subitems, tempSubItem)
	}
	return subitems
}
func YLAOrderItemProductionToIkentooOrderItemSubitem(oproductions []ylamodels.YLAOrderItemProductionInstruction) []ylamodels.SubItems {
	subitems := []ylamodels.SubItems{}
	for j := range oproductions {
		tempSubItem := ylamodels.SubItems{}
		tempSubItem.ItemSku = "FT1"
		tempSubItem.ItemPrice = 0
		tempSubItem.ItemName = oproductions[j].ProductionValues[0].Name
		subitems = append(subitems, tempSubItem)
	}
	return subitems
}
func YLAV3OrderItemToIkentooOrderItem(oitem ylamodels.YLAOrderItem) ylamodels.YLAV3Items {
	tempItem := ylamodels.YLAV3Items{}
	tempItem.Sku = oitem.ItemSku
	tempItem.Quantity = oitem.Qty
	tempItem.CustomItemPrice = oitem.Amount
	tempItem.SubItems = YLAV3OrderItemSequenceToIkentooOrderItemSubitem(oitem.OrderItemSequences)
	tempItem.SubItems = append(tempItem.SubItems, YLAV3OrderItemProductionToIkentooOrderItemSubitem(oitem.OrderItemProductionInstructions)...)
	//tempItem.SubItems = append(tempItem.SubItems, YLAV3OrderItemInstructionsToIkentooOrderItemSubitem(oitem)...)
	return tempItem
}
func YLAV3OrderItemSequenceToIkentooOrderItemSubitem(oitemsequences []ylamodels.YLAOrderItemSequence) []ylamodels.YLAV3SubItems {
	//spew.Dump(oitemsequences)
	subitems := []ylamodels.YLAV3SubItems{}
	for j := range oitemsequences {
		tempSubItem := ylamodels.YLAV3SubItems{}
		tempSubItem.Sku = helper.IntToString(oitemsequences[j].SequenceValues[0].Sku)
		tempSubItem.CustomItemPrice = oitemsequences[j].SequenceValues[0].Price
		tempSubItem.CustomItemName = oitemsequences[j].SequenceValues[0].Name
		tempSubItem.Quantity = 1
		subitems = append(subitems, tempSubItem)
	}
	return subitems
}
func YLAV3OrderItemProductionToIkentooOrderItemSubitem(oproductions []ylamodels.YLAOrderItemProductionInstruction) []ylamodels.YLAV3SubItems {
	subitems := []ylamodels.YLAV3SubItems{}
	for j := range oproductions {
		tempSubItem := ylamodels.YLAV3SubItems{}
		tempSubItem.Sku = "FT1"
		tempSubItem.Quantity = 1
		tempSubItem.CustomItemPrice = 0
		tempSubItem.CustomItemName = oproductions[j].ProductionValues[0].Name
		subitems = append(subitems, tempSubItem)
	}
	return subitems
}
func YLAV3OrderItemInstructionsToIkentooOrderItemSubitem(oitem ylamodels.YLAOrderItem) []ylamodels.YLAV3SubItems {
	subitems := []ylamodels.YLAV3SubItems{}
	if len(oitem.Instructions) > 1 {
		tempSubItem := ylamodels.YLAV3SubItems{}
		tempSubItem.Quantity = 1
		tempSubItem.Sku = "FT1"
		tempSubItem.CustomItemPrice = 0
		tempSubItem.CustomItemName = "ITEM NOTE : " + oitem.Instructions
		subitems = append(subitems, tempSubItem)
	}
	return subitems
}

func GetYLAConsumerSpend(userid int, ikentooid int) (int, interface{}) {
	var ylaUser ylamodels.YLAUser
	var ylaGiftRedemptions []ylamodels.YLAGiftRedemption
	var db = database.YLAConnection()
	db.Where("id = ?", userid).Where("ikentoo_business_id = ?", ikentooid).First(&ylaUser)
	if ylaUser.ID > 0 && ylaUser.IkentooID != "" {
		acc := GetAccountFromIkentooID(ikentooid)
		url := "https://integration.ikentoo.com/integration/1/consumers/" + ylaUser.IkentooID + "/purchaseStats?businessId=" + helper.IntToString(ikentooid)
		req, _ := http.NewRequest("GET", url, nil)
		//generic.YLANotifyOnError(err)
		req.Header.Set("Content-Type", "application/json")
		req.SetBasicAuth(acc.AuthUsername, acc.AuthPassword)
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return 400, err
		}
		body, _ := ioutil.ReadAll(resp.Body)
		i := ylamodels.YLAPurchaseStats{}
		_ = json.Unmarshal(body, &i)
		i.TotalSpend = i.PurchaseAmount
		//generic.YLANotifyOnError(err)
		db.Where("user_id = ?", ylaUser.GUID).Find(&ylaGiftRedemptions)
		//i.Claims = ylaGiftRedemptions
		for l := range ylaGiftRedemptions {
			var g ylamodels.YLAOfferSpending
			db.Where("id = ?", ylaGiftRedemptions[l].GiftID).First(&g)
			if g.Price > 0 {
				i.PurchaseAmount = i.PurchaseAmount - g.Price
				i.Deductions = append(i.Deductions, int(g.Price))
			}
		}
		return 200, i
	} else {
		return 400, "Failed"
	}
}

func ProcessAdhocPayfast(adhoc ylamodels.YLAAdHoc) (int, interface{}) {
	var db = database.YLAConnection()
	adhocdb := ylamodels.PayfastAdhoc{}
	url := adhoc.URL
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(adhoc.Body)))
	fmt.Println("HTTP : POST : " + url)
	if err != nil {
		fmt.Println(err)
	}
	adhocdb.MerchantID = helper.IntToString(adhoc.MerchantID)
	adhocdb.Description = adhoc.URL
	adhocdb.Signature = adhoc.Signature
	adhocdb.Token = adhoc.Token

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("merchant-id", helper.IntToString(adhoc.MerchantID))
	req.Header.Add("signature", adhoc.Signature)
	req.Header.Add("timestamp", adhoc.Timestamp)
	req.Header.Add("version", "v1")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	body, _ := ioutil.ReadAll(resp.Body)
	var i interface{}
	err = json.Unmarshal(body, &i)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("==================================HTTP==========================")
	spew.Dump(string(body))
	fmt.Println("==================================HTTP==========================")
	adhocdb.Response = string(body)
	db.Create(&adhocdb)
	return 200, i
}
