package ylaactions

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/reverent911/kds-golang-lib/pkg/yla/ylamodels"
	"gorm.io/gorm"
)

func YLAGetUserByID(userid int, db *gorm.DB) ylamodels.YLAUser {
	data := ylamodels.YLAUser{}
	db.Where("id = ?", userid).First(&data)
	return data
}

// YLACreateOrder ... Save Order
func YLACreateOrder(order ylamodels.YLAOrder, db *gorm.DB) ylamodels.YLAOrder {
	db.Create(&order)
	return order
}

func YLASaveCallback(callback ylamodels.YLAOrderCallback, db *gorm.DB) {
	ikmessage := ylamodels.YLAIkentooMessage{}
	ikmessage.AccountID = callback.AccountID
	ikmessage.AccountIdentifier = callback.AccountIdentifier
	ikmessage.BusinessLocationID = callback.BusinessLocationID
	ikmessage.Message = callback.Message
	ikmessage.OrderID = callback.OrderID
	ikmessage.Origin = callback.Origin
	ikmessage.Status = callback.Status
	db.Create(&ikmessage)
}

func YLASaveV3Callback(callback ylamodels.YLAV3OrderCallback, db *gorm.DB) {
	ikmessage := ylamodels.YLAIkentooMessage{}
	ikmessage.AccountID = callback.AccountIdentifier
	ikmessage.AccountIdentifier = callback.AccountIdentifier
	ikmessage.BusinessLocationID = callback.AccountIdentifier
	ikmessage.Message = callback.Reason
	ikmessage.OrderID = callback.OrderID
	ikmessage.Origin = "ONLINE"
	ikmessage.Status = callback.Status
	db.Create(&ikmessage)
}

func YLAHandleV3OneSignal(callback ylamodels.YLAV3OrderCallback, db *gorm.DB) {
	order := YLAGetFullOrderFromReference(callback.OrderID)
	callb := ylamodels.YLAOrderCallback{}
	callb.AccountID = callback.AccountIdentifier
	callb.AccountIdentifier = callback.AccountIdentifier
	callb.Message = callback.Reason
	callb.OrderID = callback.OrderID
	callb.Origin = "ONLINE"
	callb.Status = callback.Status
	callb.BusinessLocationID = order.BranchID
	YLAHandleOneSignal(callb)
}

func IkentooGetCollectionCode(callback ylamodels.YLAOrderCallback, account ylamodels.YLAAccount) (ylamodels.YLAIkentooOrder, error) {
	url := "https://integration.ikentoo.com/integration/1/order/" + callback.OrderID + "?businessLocationId=" + callback.BusinessLocationID
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println(err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(account.AuthUsername, account.AuthPassword)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err.Error())
	}
	body, _ := ioutil.ReadAll(resp.Body)
	var i ylamodels.YLAIkentooOrder
	err = json.Unmarshal(body, &i)
	return i, err
}

func YLAGetUserPastOrders(userId int, ikentooId int, db *gorm.DB) []ylamodels.YLAPastOrder {
	var orders []ylamodels.YLAPastOrder
	db.Where("user_id = ?", userId).Where("ikentoo_status IN (?)", []string{"closed", "in_production", "in_delivery", "ready_for_pickup", "accepted", "delivered"}).Order("id desc").Limit(5).Find(&orders)
	for i := range orders {
		subQuery1 := db.Select("item_sku").Where("order_id = ?", orders[i].ID).Where("ikentoo_business_id = ?", ikentooId).Table("order_items")
		db.Where("sku IN (?)", subQuery1).Where("ikentoo_business_id = ?", ikentooId).Find(&orders[i].Items)
	}
	return orders
}
