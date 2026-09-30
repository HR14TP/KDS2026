package ikentooactions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/reverent911/kds-golang-lib/pkg/generic/database"
	"github.com/reverent911/kds-golang-lib/pkg/generic/errors"
	"github.com/reverent911/kds-golang-lib/pkg/generic/helper"
	"github.com/reverent911/kds-golang-lib/pkg/ikentoo/ikentoomodels"
	"github.com/reverent911/kds-golang-lib/pkg/yla/ylaactions"
	"github.com/reverent911/kds-golang-lib/pkg/yla/ylamodels"
)

func HandleIkentooErrors(err error) {
	if err != nil {
		errors.CaptureSentryError(errors.GetSentryServices().IKENTOO_API, err)
	}
}

func ITNCreate(q map[string][]string) {

	branch := ylamodels.YLABranch{}
	itn := ylamodels.PaymentITN{}
	theorder := ylamodels.YLAOrder{}

	values := make(map[string]string)
	for key := range q {
		values[key] = strings.Join(q[key], "")
	}
	js, err := json.Marshal(values)
	if err != nil {
		panic(err)
	}

	json.Unmarshal(js, &itn)
	spew.Dump(itn)

	db := database.YLAConnection()

	db.Where("pf_merchant_id = ?", itn.MerchantID).First(&branch)
	spew.Dump(branch)
	if branch.ID > 0 {
		itn.IkentooID = branch.IkentooBusinessID
	}
	db.Create(&itn)
	if strings.Contains(itn.MPaymentID, "_") {
		s := strings.Split(itn.MPaymentID, "_")
		cardid := s[0]
		if cardid == "ORDER" {

			reference := s[1]
			db.Where("external_reference = ?", reference).First(&theorder)
			if theorder.ID > 0 {
				pfstatus := strings.ToLower(itn.PaymentStatus)
				theorder.PfStatus = pfstatus
				db.Save(&theorder)
			}
		} else {
			userID := s[1]
			card := ylamodels.YLAUserCard{}
			db.Where("id = ?", cardid).Where("user_id = ?", userID).First(&card)
			card.Token = itn.Token
			db.Save(&card)
		}
	}
	if theorder.ID > 0 {
		d := &ylamodels.YLAOrderInstruction{}
		d.OrderID = theorder.ExternalReference
		time.Sleep(60 * time.Second)
		YLAInjectOrder(*d)
	}

}

func ITNCreateZapper(itn ylamodels.PaymentITN, orderid int) {
	db := database.YLAConnection()
	db.Create(&itn)
	//fullorder := ylaactions.GetOrderDetailsFromID(orderid)
	theorder := &ylamodels.YLAOrder{}
	db.Where("id = ?", orderid).First(&theorder)
	if theorder.ID > 0 {
		theorder.PfStatus = itn.PaymentStatus
		db.Save(&theorder)
	}
	if theorder.ID > 0 {
		d := &ylamodels.YLAOrderInstruction{}
		d.OrderID = theorder.ExternalReference
		//time.Sleep(10 * time.Second)
		YLAInjectOrder(*d)
	}
}

func YLAInjectOrder(data ylamodels.YLAOrderInstruction) (int, interface{}) {
	theOrder := ylaactions.YLAGetFullOrderFromReference(data.OrderID)
	if len(theOrder.IkentooStatus) < 2 && len(theOrder.IkentooCollectionCode) < 2 {
		if theOrder.ID > 0 {
			if theOrder.Branch.ID > 0 {
				if theOrder.Branch.IkentooVersion == 2 {
					return YLAV1OrderInjection(data, theOrder)
				} else if theOrder.Branch.IkentooVersion == 3 {
					return YLAV3OrderInjection(data, theOrder)
				}
			}
		}
	}
	return 400, nil
}

func YLAV1OrderInjection(data ylamodels.YLAOrderInstruction, theOrder ylamodels.YLAOrder) (int, interface{}) {
	//spew.Dump("V1 ORDER INJECTION")
	ikentooOrder := &ylamodels.IkentooOrder{}
	var db = database.YLAConnection()
	if theOrder.ID > 0 {
		orderDetails := ylaactions.GetOrderDetailsFromID(theOrder.ID)
		theAccount := ylaactions.GetAccountFromIkentooID(helper.StringToInt(theOrder.User.IkentooBusinessID))
		ikentooOrder.BusinessLocationID = helper.StringToInt(theOrder.BranchID)
		ikentooOrder.ExternalReference = theOrder.ExternalReference
		ikentooOrder.DeliveryMode = "PICKUP"
		if len(theOrder.DeliveryType) > 0 {
			ikentooOrder.DeliveryMode = theOrder.DeliveryType
		}

		for i := range theOrder.OrderItems {
			tempItem := ylaactions.YLAOrderItemToIkentooOrderItem(theOrder.OrderItems[i])
			if len(theOrder.OrderItems[i].Instructions) > 0 {
				subItem := ylamodels.SubItems{}
				subItem.ItemPrice = 0
				subItem.ItemSku = "FT1"
				subItem.ItemName = "ITEM NOTE : " + theOrder.OrderItems[i].Instructions
				tempItem.SubItems = append(tempItem.SubItems, subItem)
			}
			ikentooOrder.Items = append(ikentooOrder.Items, tempItem)
		}

		if theOrder.PfStatus == "complete" {
			ikentooOrder.PaymentAmount = theOrder.OrderTotal
		} else if data.HasPaid {
			ikentooOrder.PaymentAmount = theOrder.OrderTotal
		} else {
			ikentooOrder.PaymentAmount = 0.0
		}

		if theOrder.User.ID > 0 {
			data.Comment = data.Comment + "|" + theOrder.User.Cell
		}

		ikentooOrder.Customer.FirstName = theOrder.User.Name
		ikentooOrder.Customer.LastName = theOrder.User.Surname

		if orderDetails.ID > 0 && orderDetails.DeliveryType != "COLLECTION" {
			useraddress := ylamodels.YLAUserAddress{}
			err := json.Unmarshal([]byte(orderDetails.DeliveryAddress), &useraddress)
			errors.YLANotifyOnError(err)
			ikentooOrder.Customer.Zip = useraddress.Postal
			if useraddress.StreetNumber > 0 {
				ikentooOrder.Customer.Address1 = helper.IntToString(useraddress.StreetNumber)
			}
			if len(useraddress.StreetName) > 0 {
				ikentooOrder.Customer.Address1 = ikentooOrder.Customer.Address1 + "," + useraddress.StreetName
			}
			ikentooOrder.Customer.Address1 = ikentooOrder.Customer.Address1 + "|" + theOrder.User.Cell
			ikentooOrder.Customer.Address2 = useraddress.Address + "," + useraddress.BuildingName
			ikentooOrder.Customer.City = useraddress.Town
			ikentooOrder.Customer.Country = "South Africa"
		}

		ikentooOrder.Customer.Email = theOrder.User.Email
		ikentooOrder.Customer.ExternalReference = "CO" + theOrder.User.GUID

		if orderDetails.OrderNotes != "null" && len(orderDetails.OrderNotes) > 0 && len(orderDetails.DeliveryType) > 0 {
			ikentooOrder.Comment = "***" + orderDetails.DeliveryType + "|" + orderDetails.OrderNotes + "***"
			tempItem := ylamodels.OrderItemIkentoo{}
			tempItem.ItemPrice = 0
			tempItem.ItemSku = "FT1"
			tempItem.Quantity = 1
			tempItem.ItemName = "ORDER NOTE : " + orderDetails.OrderNotes
			ikentooOrder.Items = append(ikentooOrder.Items, tempItem)
		} else {
			ikentooOrder.Comment = "***" + ikentooOrder.DeliveryMode + "|" + theOrder.User.Cell + "***"
		}

		if ikentooOrder.DeliveryMode == "COLLECTION" {
			ikentooOrder.DeliveryMode = "PICKUP"
		}

		theOrderString, _ := json.Marshal(ikentooOrder)
		jsonbody, err := json.Marshal(ikentooOrder)
		errors.YLASentryCapture(err)
		ikorder := &ylamodels.IkentooInjection{}
		ikorder.Theorder = string(theOrderString)
		ikorder.Injection = string(jsonbody)
		db.Create(ikorder)

		url := "https://integration.ikentoo.com/integration/1/order"

		if err == nil {
			req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonbody))
			errors.YLASentryCapture(err)
			req.Header.Set("Content-Type", "application/json")
			req.SetBasicAuth(theAccount.AuthUsername, theAccount.AuthPassword)
			client := &http.Client{}
			resp, err := client.Do(req)
			errors.YLASentryCapture(err)
			if err != nil {
				return 400, err
			}
			body, _ := ioutil.ReadAll(resp.Body)
			ikorder.Response = string(body)
			db.Save(ikorder)
			var i interface{}
			err = json.Unmarshal(body, &i)
			errors.YLASentryCapture(err)
			return resp.StatusCode, i
		}
	}
	return 200, ikentooOrder
}

// V3 NEW API INJECTION
// 8e121e62-c4c4-4bfa-80ab-e2ec5514fae0
func YLAV3OrderInjection(data ylamodels.YLAOrderInstruction, theOrder ylamodels.YLAOrder) (int, interface{}) {
	//spew.Dump("V3 ORDER INJECTION")
	//theOrder := ylaactions.YLAGetFullOrderFromReference(data.OrderID)
	ikentooOrder := &ylamodels.YLAV3Order{}
	var db = database.YLAConnection()
	if theOrder.ID > 0 {
		orderDetails := ylaactions.GetOrderDetailsFromID(theOrder.ID)
		ikentooOrder.BusinessLocationID = helper.StringToInt(theOrder.BranchID)
		ikentooOrder.ThirdPartyReference = theOrder.ExternalReference
		ikentooOrder.EndpointID = "YLAV3CALLBACK"
		ikentooOrder.AccountProfileCode = "ONLINE_PICKUP"

		for i := range theOrder.OrderItems {
			//check if item processing is a STAFF TIP ITEM
			if theOrder.OrderItems[i].ItemSku == theOrder.Branch.TipSKU {
				//THIS IS A TIP ITEM
				ikentooOrder.Payment.TipAmount = theOrder.OrderItems[i].Amount
			} else {
				tempItem := ylaactions.YLAV3OrderItemToIkentooOrderItem(theOrder.OrderItems[i])
				if len(theOrder.OrderItems[i].Instructions) > 0 {
					subItem := ylamodels.YLAV3SubItems{}
					subItem.CustomItemPrice = 0
					subItem.Sku = "FT1"
					subItem.Quantity = 1
					subItem.CustomItemName = "ITEM NOTE : " + theOrder.OrderItems[i].Instructions
					tempItem.SubItems = append(tempItem.SubItems, subItem)
				}
				ikentooOrder.Items = append(ikentooOrder.Items, tempItem)
			}
		}

		if theOrder.PfStatus == "complete" {
			ikentooOrder.Payment.PaymentMethod = "ONLINE"
			ikentooOrder.Payment.PaymentAmount = theOrder.OrderTotal
		} else if data.HasPaid {
			ikentooOrder.Payment.PaymentMethod = "ONLINE"
			ikentooOrder.Payment.PaymentAmount = theOrder.OrderTotal
		}

		ikentooOrder.CustomerInfo.FirstName = theOrder.User.Name
		ikentooOrder.CustomerInfo.LastName = theOrder.User.Surname
		ikentooOrder.CustomerInfo.Email = theOrder.User.Email
		ikentooOrder.CustomerInfo.ContactNumberAsE164 = theOrder.User.Cell

		if orderDetails.ID > 0 && orderDetails.DeliveryType != "COLLECTION" {
			ikentooOrder.AccountProfileCode = "ONLINE_DELIVERY"
			useraddress := ylamodels.YLAUserAddress{}
			err := json.Unmarshal([]byte(orderDetails.DeliveryAddress), &useraddress)
			errors.YLANotifyOnError(err)
			ikentooOrder.DeliveryAddress.Zip = useraddress.Postal
			if useraddress.StreetNumber > 0 {
				ikentooOrder.DeliveryAddress.AddressLine1 = helper.IntToString(useraddress.StreetNumber)
			}
			if len(useraddress.StreetName) > 0 {
				ikentooOrder.DeliveryAddress.AddressLine1 = ikentooOrder.DeliveryAddress.AddressLine1 + "," + useraddress.StreetName
			}
			ikentooOrder.DeliveryAddress.AddressLine1 = ikentooOrder.DeliveryAddress.AddressLine1 + "|" + theOrder.User.Cell
			ikentooOrder.DeliveryAddress.AddressLine2 = useraddress.Address + "," + useraddress.BuildingName
			ikentooOrder.DeliveryAddress.City = useraddress.Town
			ikentooOrder.TakeAway = true
		}

		if orderDetails.OrderNotes != "null" && len(orderDetails.OrderNotes) > 0 && len(orderDetails.DeliveryType) > 0 {
			ikentooOrder.OrderNote = "***" + orderDetails.DeliveryType + "|" + orderDetails.OrderNotes + "***"
			tempItem := ylamodels.YLAV3Items{}
			tempItem.CustomItemPrice = 0
			tempItem.Sku = "FT1"
			tempItem.Quantity = 1
			tempItem.CustomItemName = "ORDER NOTE : " + orderDetails.OrderNotes
			ikentooOrder.Items = append(ikentooOrder.Items, tempItem)
		} else {
			ikentooOrder.OrderNote = "***" + orderDetails.DeliveryType + "|" + theOrder.User.Cell + "***"
		}
		ikentooOrder.ClientCount = 1
		ikentooOrder.MaxTimeToAttemptOrderDeliverToPos = 1800000

		theOrderString, _ := json.Marshal(ikentooOrder)
		jsonbody, err := json.Marshal(ikentooOrder)
		ikorder := &ylamodels.IkentooInjection{}
		ikorder.Theorder = string(theOrderString)
		ikorder.Injection = string(jsonbody)
		db.Create(ikorder)
		url := "https://api.ikentoo.com/o/op/1/order/toGo"
		var bearer = "Bearer 8e121e62-c4c4-4bfa-80ab-e2ec5514fae0"
		spew.Dump(ikorder)

		if err == nil {
			req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonbody))
			errors.YLASentryCapture(err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Add("Authorization", bearer)
			client := &http.Client{}
			resp, err := client.Do(req)
			errors.YLASentryCapture(err)
			if err != nil {
				return 400, err
			}
			body, _ := ioutil.ReadAll(resp.Body)
			ikorder.Response = string(body)
			db.Save(ikorder)
			i := ylamodels.YLAV3INJECTION{}
			err = json.Unmarshal(body, &i)
			errors.YLASentryCapture(err)
			if i.Status == "ok" {
				i.Status = "OK"
			}
			spew.Dump(i)
			return resp.StatusCode, i
		}

		return 200, ikentooOrder
	}
	return 200, ikentooOrder
}

// GetMenuFromID ... get V3 menu from ID
func GetMenuFromID(token string, menuid string, locationid string) ikentoomodels.MainMenu {
	url := "https://api.ikentoo.com/o/op/1/menu/load/" + menuid + "?businessLocationId=" + locationid
	var bearer = "Bearer " + token
	req, err := http.NewRequest("GET", url, nil)
	HandleIkentooErrors(err)
	req.Header.Add("Authorization", bearer)
	client := &http.Client{}
	resp, err := client.Do(req)
	HandleIkentooErrors(err)
	body, _ := ioutil.ReadAll(resp.Body)
	menu := ikentoomodels.MainMenu{}
	err = json.Unmarshal([]byte(body), &menu)
	HandleIkentooErrors(err)
	return menu
}

// GetRichItems ... Get Rich Items
func GetRichItems(token string, businessid string, page int) ikentoomodels.RichItemResponse {
	url := "https://api.ikentoo.com/i/richItem/" + businessid + "?size=100&page=" + helper.IntToString(page) + "&sort=sku"
	var bearer = "Bearer " + token
	fmt.Println(url)
	req, err := http.NewRequest("GET", url, nil)
	HandleIkentooErrors(err)
	req.Header.Add("Authorization", bearer)
	client := &http.Client{}
	resp, err := client.Do(req)
	HandleIkentooErrors(err)
	body, _ := ioutil.ReadAll(resp.Body)
	items := ikentoomodels.RichItemResponse{}
	err = json.Unmarshal([]byte(body), &items)
	HandleIkentooErrors(err)
	//spew.Dump(items)
	return items
}
