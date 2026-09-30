package ylaactions

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/reverent911/kds-golang-lib/pkg/generic/database"
	"github.com/reverent911/kds-golang-lib/pkg/generic/errors"
	"github.com/reverent911/kds-golang-lib/pkg/generic/helper"
	"github.com/reverent911/kds-golang-lib/pkg/yla/ylamodels"
	"gorm.io/gorm"
)

type PayfastTokenPaymentResponse struct {
	Code   int                                 `json:"code"`
	Status string                              `json:"status"`
	Data   PayfastTokenPaymentResponseResponse `json:"data"`
}
type PayfastTokenPaymentResponseResponse struct {
	Response interface{} `json:"response"`
	Message  string      `json:"message"`
}
type PayfastTokenPaymentResponseResponseData struct {
	Code   string `json:"code"`
	Reason string `json:"reason"`
}

// {"code":200,"status":"success","data":{"response":{"status":1,"status_reason":"","status_text":"ACTIVE"}}}
type PayfastTokenResponse struct {
	Code   int                          `json:"code"`
	Status string                       `json:"status"`
	Data   PayfastTokenResponseResponse `json:"data"`
}

type PayfastTokenResponseResponse struct {
	Response PayfastTokenResponseResponseData `json:"response"`
}
type PayfastTokenResponseResponseData struct {
	Status       *int      `json:"status"`
	StatusReason *string   `json:"status_reason"`
	StatusText   *string   `json:"status_text"`
	GUID         *[]string `json:"guid"`
	Message      *string   `json:"message"`
}
type PayfastSignature struct {
	Signature string
	Builder   string
	Headers   map[string]string
}

func (p *PayfastSignature) Add(key string, value string) {
	p.Builder += key
	p.Builder += "="
	p.Builder += url.QueryEscape(value)
	p.Builder += "&"
	if p.Headers == nil {
		p.Headers = make(map[string]string)
	}
	p.Headers[key] = value
}
func (p *PayfastSignature) AddHeaders(req *http.Request) {
	for index, field := range p.Headers {
		if index != "passphrase" && index != "amount" && index != "item_name" && index != "item_description" && index != "m_payment_id" && index != "cc_cvv" {
			req.Header.Add(index, field)
		}
	}
}
func (p *PayfastSignature) Generate(passphrase string) {
	URLString := strings.TrimSuffix(p.Builder, "&")
	p.Signature = GetMD5Hash(URLString)
	if p.Headers == nil {
		p.Headers = make(map[string]string)
	}
	p.Headers["signature"] = p.Signature
}

func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func payfastCancelToken(card ylamodels.YLAUserCard, db *gorm.DB) {
	fmt.Println("Cancel Card Subscription")
	var zaTime, _ = time.LoadLocation("Africa/Johannesburg")
	if len(card.Token) > 0 {
		branch := YLALambdaFindBranchID(card.BranchID, db)
		if branch.ID < 1 {
			user := YLANewFindUserOnID(card.UserID, db)
			branch = YLALambdaFindBranchByBusiness(user.IkentooBusinessID, db)
		}
		if branch.ID > 0 {
			i := PayfastTokenResponse{}
			p := PayfastSignature{}
			p.Add("merchant-id", branch.PfMerchantID)
			p.Add("passphrase", branch.PfPassphrase)
			p.Add("timestamp", time.Now().In(zaTime).Format(time.RFC3339))
			p.Add("version", "v1")
			p.Generate(branch.PfPassphrase)
			//branch is set , lets check this token
			url := fmt.Sprintf("https://api.payfast.co.za/subscriptions/%s/cancel", string(card.Token))
			fmt.Println(url)
			req, err := http.NewRequest("PUT", url, nil)
			if err != nil {
				panic(err)
			}
			p.AddHeaders(req)
			fmt.Println(req.Header)
			fmt.Println(p.Builder)
			client := &http.Client{}
			resp, err := client.Do(req)
			errors.YLASentryCapture(err)
			if err != nil {
				fmt.Println(err)
			}
			body, _ := ioutil.ReadAll(resp.Body)
			fmt.Println(string(body))
			err = json.Unmarshal(body, &i)
			errors.YLASentryCapture(err)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println(i)
		}
	} else {
		fmt.Println("No TOKEN - No Need To Cancel")
	}
}

func PayfastChargeToken(orderid int, cardid int, userid int) (int, interface{}) {
	var zaTime, _ = time.LoadLocation("Africa/Johannesburg")
	var db = database.YLADebuggedConnection()
	user := YLAGetUserByID(userid, db)
	card := YLALambdaGetCard(cardid, db)
	order := YLAGetFullOrderFromID(orderid)
	i := PayfastTokenPaymentResponse{}
	fmt.Println(order.OrderTotal)
	if user.ID > 0 && card.ID > 0 && order.ID > 0 {
		if user.ID == card.UserID && len(card.Token) > 0 {
			p := PayfastSignature{}
			form := url.Values{}
			db.Model(&ylamodels.YLAOrder{}).Where("id = ?", order.ID).Update("order_status", 7)

			//p.Add("amount", helper.FloatToString(order.OrderTotal))
			p.Add("amount", helper.IntToString(int(order.OrderTotal*100)))
			form.Add("amount", helper.IntToString(int(order.OrderTotal*100)))

			p.Add("item_description", "ORDER_"+order.ExternalReference)
			form.Add("item_description", "ORDER_"+order.ExternalReference)

			p.Add("item_name", "ORDER_"+order.ExternalReference)
			form.Add("item_name", "ORDER_"+order.ExternalReference)

			p.Add("m_payment_id", "ORDER_"+order.ExternalReference)
			form.Add("m_payment_id", "ORDER_"+order.ExternalReference)

			p.Add("merchant-id", order.Branch.PfMerchantID)

			p.Add("passphrase", order.Branch.PfPassphrase)
			p.Add("timestamp", time.Now().In(zaTime).Format(time.RFC3339))
			p.Add("version", "v1")
			p.Generate(order.Branch.PfPassphrase)
			fmt.Println(p)
			url := fmt.Sprintf("https://api.payfast.co.za/subscriptions/%s/adhoc", string(card.Token))
			req, err := http.NewRequest("POST", url, strings.NewReader(form.Encode()))
			if err != nil {
				panic(err)
			}
			p.AddHeaders(req)
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			client := &http.Client{}
			resp, err := client.Do(req)
			errors.YLASentryCapture(err)
			if err != nil {
				fmt.Println(err)
			}
			body, _ := ioutil.ReadAll(resp.Body)
			fmt.Println(string(body))
			err = json.Unmarshal(body, &i)
			errors.YLASentryCapture(err)
			if err != nil {
				fmt.Println(err)
			}
			if i.Code != 200 {
				return i.Code, i.Data
			}

			if strings.Contains(i.Data.Message, "Transaction was successful") {
				//Success
				fmt.Println(i.Data.Message)
				db.Model(&ylamodels.YLAOrder{}).Where("id = ?", order.ID).Update("order_status", 11)
				return i.Code, i.Data.Message
			}
			if reflect.TypeOf(i.Data.Response).Kind().String() == "map" {
				//there was an error
				j := PayfastTokenPaymentResponseResponseData{}
				jsonbody, err := json.Marshal(i.Data.Response)
				db.Model(&ylamodels.YLAOrder{}).Where("id = ?", order.ID).Update("order_status", 10)
				if err != nil {
					return 400, "Payment Failed"
				}
				err = json.Unmarshal(jsonbody, &j)
				if err != nil {
					return 400, "Payment Failed"
				}
				//fmt.Println(j.Reason)
				return 400, j.Reason
			}
		}
	}
	return 400, "Payment Failed"
}

func payfastCheckToken(cards []ylamodels.YLAUserCard, db *gorm.DB) []ylamodels.YLAUserCard {
	var zaTime, _ = time.LoadLocation("Africa/Johannesburg")
	for index, card := range cards {
		if len(card.Token) > 0 {
			branch := YLALambdaFindBranchID(card.BranchID, db)
			if branch.ID < 1 {
				user := YLANewFindUserOnID(card.UserID, db)
				branch = YLALambdaFindBranchByBusiness(user.IkentooBusinessID, db)
			}
			if branch.ID > 0 {
				i := PayfastTokenResponse{}
				p := PayfastSignature{}
				p.Add("merchant-id", branch.PfMerchantID)
				p.Add("passphrase", branch.PfPassphrase)
				p.Add("timestamp", time.Now().In(zaTime).Format(time.RFC3339))
				p.Add("version", "v1")
				p.Generate(branch.PfPassphrase)
				//branch is set , lets check this token
				url := fmt.Sprintf("https://api.payfast.co.za/subscriptions/%s/fetch", string(card.Token))
				fmt.Println(url)
				req, err := http.NewRequest("GET", url, nil)
				if err != nil {
					panic(err)
				}
				p.AddHeaders(req)
				fmt.Println(req.Header)
				fmt.Println(p.Builder)
				client := &http.Client{}
				resp, err := client.Do(req)
				errors.YLASentryCapture(err)
				if err != nil {
					fmt.Println(err)
				}
				body, _ := ioutil.ReadAll(resp.Body)
				fmt.Println(string(body))
				err = json.Unmarshal(body, &i)
				errors.YLASentryCapture(err)
				if err != nil {
					fmt.Println(err)
				}
				fmt.Println(i)
				if i.Code == 200 && *i.Data.Response.StatusText == "ACTIVE" {
					//Card is Active
					fmt.Println("Card is Active and Ready")
				} else {
					//Card is InActive
					fmt.Println("Card is InActive - Wiping Token")
					cards[index].Token = ""
					db.Save(&cards[index])
				}
			}
		}
		fmt.Println(index, card)
	}
	return cards
}

func runTestHanno() {
	userid := 2742
	cardid := 5853
	orderid := 58037
	fmt.Println(PayfastChargeToken(orderid, cardid, userid))
}
