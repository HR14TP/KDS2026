package ylaactions

import (
	"fmt"

	"github.com/reverent911/kds-golang-lib/pkg/generic/database"
	"github.com/reverent911/kds-golang-lib/pkg/generic/helper"
	"github.com/reverent911/kds-golang-lib/pkg/yla/ylamodels"
	"github.com/tbalthazar/onesignal-go"
)

var MESSAGE_IN_PRODUCTION = "Your order has been accepted and we are preparing it, please make sure your notifications are enabled for future updates on your order. For any inquiries your order reference is #"
var MESSAGE_READY_FOR_PICKUP = "Your order is Ready! If you ordered for Collection you can collect from the store, alternatively if you ordered for Delivery your order will be delivered to you soon!. For any inquiries your order reference is #"
var MESSAGE_IN_DELIVERY = "Your order is Ready for Pickup if you selected collection at checkout, if you selected Delivery Your order is on the way to you, please keep an eye out for our delivery guy. For any inquiries your order reference is #"
var MESSAGE_REJECTED = "Oh No! There was an error placing your order , please get in contact with the store as soon as possible or use the Feedback option on the App under the More tab at the bottom."
var MESSAGE_DELIVERED = "Thank you for your order, did you enjoy your meal ? Give us a rating (1-5) and any feedback."

func YLAHandleOneSignal(callback ylamodels.YLAOrderCallback) {
	var db = database.YLAConnection()
	order := YLAGetFullOrderFromReference(callback.OrderID)
	if order.ID > 0 {
		user := YLAGetUserByID(order.UserID, db)
		if user.ID > 0 {
			account := GetAccountFromIkentooID(helper.StringToInt(user.IkentooBusinessID))
			ikorder, _ := IkentooGetCollectionCode(callback, account)
			if len(ikorder.CollectionCode) > 0 && len(order.IkentooCollectionCode) < 1 {
				order.IkentooCollectionCode = ikorder.CollectionCode
				db.Save(&order)
			}
			if account.ID > 0 && account.OnesignalID != "1111" {
				fmt.Println("Sending Onesignal")
				client := onesignal.NewClient(nil)
				client.UserKey = "ZGY5MjgwZWUtOTE1Ni00NDZlLTg5NDUtMWUxODA5NWJlMDdl"
				client.AppKey = account.OnesignalKey
				fmt.Println("Sending Onesignal #2")
				clientmessage := map[string]string{"en": "English message"}
				sendmessage := false
				if callback.Status == "IN_PRODUCTION" {
					clientmessage = map[string]string{"en": MESSAGE_IN_PRODUCTION + order.IkentooCollectionCode}
					sendmessage = true
				} else if callback.Status == "READY_FOR_PICKUP" {
					clientmessage = map[string]string{"en": MESSAGE_READY_FOR_PICKUP + order.IkentooCollectionCode}
					sendmessage = true
				} else if callback.Status == "IN_DELIVERY" {
					clientmessage = map[string]string{"en": MESSAGE_IN_DELIVERY + order.IkentooCollectionCode}
					sendmessage = true
				} else if callback.Status == "REJECTED" || callback.Status == "FAILED" {
					clientmessage = map[string]string{"en": MESSAGE_REJECTED}
					sendmessage = true
				} else if callback.Status == "DELIVERED" {
					clientmessage = map[string]string{"en": MESSAGE_DELIVERED}
					sendmessage = true
				}
				if sendmessage {
					fmt.Println("Sending Onesignal #3")
					if len(user.OsUserID) > 0 {
						notificationReq := &onesignal.NotificationRequest{
							AppID:            account.OnesignalID,
							Contents:         clientmessage,
							IsIOS:            true,
							IncludePlayerIDs: []string{user.OsUserID},
						}
						createRes, res, err := client.Notifications.Create(notificationReq)
						usernotification := ylamodels.YLAUserNotification{}
						usernotification.IkentooID = helper.StringToInt(user.IkentooBusinessID)
						usernotification.Message = clientmessage["en"]
						usernotification.UserID = user.ID
						if err != nil {
							usernotification.Response += "|" + err.Error()
						} else {
							if createRes != nil {
								usernotification.Response = createRes.ID + "|" + helper.IntToString(res.StatusCode)
							} else {
								usernotification.Response = helper.IntToString(res.StatusCode)
							}
						}
						fmt.Println("Sending Onesignal #4")
						db.Create(&usernotification)
					} else {
						fmt.Println("Sending Onesignal #5 - No Player ID")
					}
				}
			}
		}
	}
}

func YLALoyaltyNotifify(offer ylamodels.YLAOffer, userid int) {
	var db = database.YLAConnection()
	user := YLAGetUserByID(userid, db)
	if user.ID > 0 {
		account := GetAccountFromIkentooID(helper.StringToInt(user.IkentooBusinessID))
		if account.ID > 0 && account.OnesignalID != "1111" {
			fmt.Println("Sending Onesignal #1")
			client := onesignal.NewClient(nil)
			client.UserKey = "ZGY5MjgwZWUtOTE1Ni00NDZlLTg5NDUtMWUxODA5NWJlMDdl"
			client.AppKey = account.OnesignalKey
			fmt.Println("Sending Onesignal #2")
			clientmessage := map[string]string{"en": "English message"}
			sendmessage := false
			if offer.Type == "gift" {
				sendmessage = true
				clientmessage = map[string]string{"en": "Congratulations! You have just claimed " + offer.Title + " from the store. Thank you for your support!"}
			}
			if offer.Type == "USERCLAIM" {
				sendmessage = true
				clientmessage = map[string]string{"en": "Congratulations! You have just used " + helper.FloatToString(offer.Price) + " loyalty points for a discount. Thank you for your support!"}
			}
			if sendmessage && len(user.OsUserID) > 0 {
				notificationReq := &onesignal.NotificationRequest{
					AppID:            account.OnesignalID,
					Contents:         clientmessage,
					IsIOS:            true,
					IncludePlayerIDs: []string{user.OsUserID},
				}
				createRes, res, err := client.Notifications.Create(notificationReq)
				usernotification := ylamodels.YLAUserNotification{}
				usernotification.IkentooID = helper.StringToInt(user.IkentooBusinessID)
				usernotification.Message = clientmessage["en"]
				usernotification.UserID = user.ID
				if err != nil {
					usernotification.Response += "|" + err.Error()
				} else {
					if createRes != nil {
						usernotification.Response = createRes.ID + "|" + helper.IntToString(res.StatusCode)
					} else {
						usernotification.Response = helper.IntToString(res.StatusCode)
					}
				}
				fmt.Println("Sending Onesignal #4")
				db.Create(&usernotification)
			} else {
				fmt.Println("Sending Onesignal #5 - No Player ID")
			}
		}
	}
}
