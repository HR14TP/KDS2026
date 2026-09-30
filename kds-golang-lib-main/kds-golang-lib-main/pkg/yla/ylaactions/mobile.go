package ylaactions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"strings"

	"github.com/reverent911/kds-golang-lib/pkg/generic/database"
	"github.com/reverent911/kds-golang-lib/pkg/generic/errors"
	"github.com/reverent911/kds-golang-lib/pkg/generic/helper"
	"github.com/reverent911/kds-golang-lib/pkg/yla/ylamodels"
	uuid "github.com/satori/go.uuid"
	"gorm.io/gorm"
)

// YLALambdaUserMenuFromBaseLite ... User Menu From Base Lite
func YLALambdaUserMenuFromBaseLite(id int, clientid int, userid int, db *gorm.DB) (int, interface{}) {

	if db == nil {
		db = database.YLADebuggedConnection()
	}

	/* need to check if user is part of a membership */
	var hasmembership = false
	usr := ylamodels.YLAUser{}
	ms := ylamodels.YLAMembership{}

	if userid > 0 {
		db.Where("id = ?", userid).First(&usr)
		if usr.ID > 0 && usr.MembershipID > 0 {
			/* user is a member of a group */
			db.Where("id = ?", usr.MembershipID).First(&ms)
			if ms.ID > 0 && ms.Type != "PRICE" {
				hasmembership = true
			}
		}
	}

	data := ylamodels.YLAMenu{}
	db.Where("id = ?", id).First(&data)
	if data.ID > 0 {
		db.Where("parent_screen_id = ?", data.ID).Where("deleted_at IS NULL").Order("order_no asc").Find(&data.Menus)
		/*V2*/
		subQuery := db.Select("item_id").Where("menu_id = ?", data.ID).Where("ikentoo_business_id = ?", clientid).Table("menu_items")
		db.Where("sku IN (?)", subQuery).Where("ikentoo_business_id = ?", clientid).Where("disabled = ?", 0).Find(&data.Items)
		for i := range data.Items {
			if hasmembership {
				discount := helper.StringToFloat(ms.Type)
				price := data.Items[i].CostPrice
				val := price - ((price * discount) / 100)
				data.Items[i].CostPrice = math.Round(val)
			}
			if data.Items[i].PhotoID != "" {
				db.Where("id = ?", data.Items[i].PhotoID).Find(&data.Items[i].File)
			}

			/*V2*/
			subQuery1 := db.Select("sequence_id").Where("item_sku = ?", data.Items[i].Sku).Where("ikentoo_business_id = ?", clientid).Table("item_sequences")
			db.Where("id IN (?)", subQuery1).Where("ikentoo_business_id = ?", clientid).Where("deleted_at is NULL").Order("`order` asc").Find(&data.Items[i].Sequences)
			for j := range data.Items[i].Sequences {
				db.Where("sequence_id = ?", data.Items[i].Sequences[j].ID).Where("deleted_at IS NULL").Find(&data.Items[i].Sequences[j].SequenceValues)
			}

			/*V2*/
			subQuery2 := db.Select("production_instruction_id").Where("item_sku = ?", data.Items[i].Sku).Where("ikentoo_business_id = ?", clientid).Table("item_production_instructions")
			db.Where("id IN (?)", subQuery2).Where("ikentoo_business_id = ?", clientid).Where("deleted_at is NULL").Find(&data.Items[i].ProductionInstructions)
			for k := range data.Items[i].ProductionInstructions {
				db.Where("production_instruction_id = ?", data.Items[i].ProductionInstructions[k].ID).Where("deleted_at IS NULL").Find(&data.Items[i].ProductionInstructions[k].ProductionInstructionValue)
			}
		}
	}
	return 200, data
}

func YLANewGetUserPastOrders(userId int, ikentooId int) []ylamodels.YLAPastOrder {
	db := database.YLADebuggedConnection()
	var orders []ylamodels.YLAPastOrder
	db.Where("user_id = ?", userId).Where("order_status NOT IN (100)").Order("id desc").Limit(20).Find(&orders)
	for i := range orders {
		subQuery1 := db.Select("item_sku").Where("order_id = ?", orders[i].ID).Where("ikentoo_business_id = ?", ikentooId).Table("order_items")
		db.Where("sku IN (?)", subQuery1).Where("ikentoo_business_id = ?", ikentooId).Find(&orders[i].Items)
	}
	database.CloseDatabase(db)
	return orders
}

func YLANewFindUserOnEmailAndCell(ikentooID int, email string, cellphone string, db *gorm.DB) ylamodels.YLAUser {
	terminate := false
	if db == nil {
		terminate = true
		db = database.YLADebuggedConnection()
	}
	var user ylamodels.YLAUser
	db.Where("ikentoo_business_id = ?", ikentooID).Where("email = ?", email).Where("cell = ?", cellphone).First(&user)
	if terminate {
		database.CloseDatabase(db)
	}
	return user
}

func YLANewFindUserOnEmail(ikentooID int, email string, db *gorm.DB) ylamodels.YLAUser {
	terminate := false
	if db == nil {
		terminate = true
		db = database.YLADebuggedConnection()
	}
	var user ylamodels.YLAUser
	db.Where("ikentoo_business_id = ?", ikentooID).Where("email = ?", email).First(&user)
	if terminate {
		database.CloseDatabase(db)
	}
	return user
}

func YLANewFindUserOnID(userID int, db *gorm.DB) ylamodels.YLAUser {
	terminate := false
	if db == nil {
		terminate = true
		db = database.YLADebuggedConnection()
	}
	var user ylamodels.YLAUser
	db.Where("id = ?", userID).First(&user)
	if terminate {
		database.CloseDatabase(db)
	}
	return user
}

func YLANewAccountFromIkentooID(id int, db *gorm.DB) ylamodels.YLAAccount {
	terminate := false
	if db == nil {
		terminate = true
		db = database.YLADebuggedConnection()
	}
	acc := ylamodels.YLAAccount{}
	db.Where("ikentoo_id = ?", id).Find(&acc)
	if terminate {
		database.CloseDatabase(db)
	}
	return acc
}

func YLANewSaveRequestWithAuth(input string, status int, response string, url string, username string, password string) {
	res := ylamodels.YLARequests{}
	res.Input = input
	res.Status = status
	res.Response = response
	res.URL = url
	res.AuthUser = username
	res.AuthPassword = password
	db := database.YLADebuggedConnection()
	db.Create(&res)
	database.CloseDatabase(db)
}

func YLANewProcessFlowUserLogin(input ylamodels.YLAUserLoginInput) (int, interface{}) {
	db := database.YLADebuggedConnection()
	ikuser := YLANewFindUserOnEmailAndCell(input.IkentooBusinessID, input.Email, input.Phone, db)
	if ikuser.ID > 0 {
		return IkentooFindAndAssignUserCard(ikuser, db)
	} else {
		return 400, "E5: User not found, please try to register."
	}
}
func YLANewProcessFlowRegisterUser(consumer ylamodels.IkentooConsumer) (int, interface{}) {
	db := database.YLADebuggedConnection()
	ikcreated := YLANewCreateIkentooUser(consumer, db)
	if helper.StringToInt(ikcreated.ID) > 0 {
		ikuser := YLANewCreateBasicUser(consumer, ikcreated, db)
		if ikuser.ID > 0 {
			return IkentooFindAndAssignUserCard(ikuser, db)
		} else {
			return 400, "Could not create your user, please contact the App Administrator with code E13"
		}
	} else {
		return 400, "Invalid data to create your user, please contact the App Administrator with code E12"
	}
}

// Check if there is an available card for this user or if the user already has an card
func YLANewCreateBasicUser(consumer ylamodels.IkentooConsumer, ikcreated ylamodels.IkentooConsumerCreated, db *gorm.DB) ylamodels.YLAUser {
	terminate := false
	if db == nil {
		terminate = true
		db = database.YLADebuggedConnection()
	}
	user := YLANewFindUserOnEmail(consumer.IkentooBusinessID, consumer.Email, db)
	if user.ID > 0 {
		user.Cell = consumer.PhoneNumber1
		user.Surname = consumer.LastName
		user.Name = consumer.FirstName
		user.IkentooID = ikcreated.ID
		//save updated fields
		db.Save(&user)
	} else {
		user.Cell = consumer.PhoneNumber1
		user.Surname = consumer.LastName
		user.Name = consumer.FirstName
		user.Email = consumer.Email
		user.IkentooBusinessID = helper.IntToString(consumer.IkentooBusinessID)
		//existing.OsPushToken = user.OsPushToken
		//existing.OsUserID = user.OsUserID
		user.IkentooID = ikcreated.ID
		user.GUID = (uuid.Must(uuid.NewV1(), nil)).String()
		user.Password = helper.AuthCreatePassword(user.GUID)
		user.UserTypeID = 2
		user.Archived = 0
		db.Create(&user)
	}
	if terminate {
		database.CloseDatabase(db)
	}
	return user
}

// Handler Application Handler
func YLANewCreateIkentooUser(consumer ylamodels.IkentooConsumer, db *gorm.DB) ylamodels.IkentooConsumerCreated {
	acc := YLANewAccountFromIkentooID(consumer.IkentooBusinessID, db)
	var resultset ylamodels.IkentooConsumerCreated
	if acc.ID > 0 {
		url := "https://integration.ikentoo.com/integration/1/consumers"
		jsonbody, err := json.Marshal(consumer)
		if err == nil {
			req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonbody))
			errors.YLANotifyOnError(err)
			req.Header.Set("Content-Type", "application/json")
			req.SetBasicAuth(acc.AuthUsername, acc.AuthPassword)
			client := &http.Client{}
			resp, err := client.Do(req)
			errors.YLANotifyOnError(err)
			body, _ := ioutil.ReadAll(resp.Body)
			err = json.Unmarshal(body, &resultset)
			errors.YLANotifyOnError(err)
			//fmt.Println(string(body))
			return resultset
		} else {
			return resultset
		}
	} else {
		return resultset
	}
}

func IkentooFindAndAssignUserCard(user ylamodels.YLAUser, db *gorm.DB) (int, interface{}) {
	if db == nil {
		db = database.YLADebuggedConnection()
	}
	card := ylamodels.YLAPunchCard{}
	db.Where("user_id = ?", user.GUID).Where("ikentoo_business_id = ?", user.IkentooBusinessID).First(&card)
	if card.ID > 0 {
		//found one lets return it
		fmt.Println("OLD CARD")
		//spew.Dump(card)
		//return IkenooCard(card, user.Email, db)
	} else {
		//lets make a new one
		db.Where("user_id = '0'").Where("ikentoo_business_id = ?", user.IkentooBusinessID).First(&card)
		if card.ID > 0 {
			fmt.Println("NEW CARD")
			//spew.Dump(card)
			card.UserID = user.GUID
			db.Save(&card)
			//return IkenooCard(card, user.Email, db)
		} else {
			return 400, "Cannot assign a new loyalty card , please contact the administrator."
		}
	}
	return IkenooCard(card, user.Email, db, user)
}

func IkenooCard(card ylamodels.YLAPunchCard, email string, db *gorm.DB, user ylamodels.YLAUser) (int, interface{}) {
	var i ylamodels.IkentooGenericResponse
	acc := YLANewAccountFromIkentooID(helper.StringToInt(card.IkentooBusinessID), db)
	if acc.ID > 0 {
		url := "https://integration.ikentoo.com/integration/1/card/" + card.UUID + "/link/" + card.IkentooBusinessLocationID + "/" + email
		//spew.Dump(url)
		req, err := http.NewRequest("PUT", url, nil)
		errors.YLANotifyOnError(err)
		req.Header.Set("Content-Type", "application/json")
		req.SetBasicAuth(acc.AuthUsername, acc.AuthPassword)
		client := &http.Client{}
		resp, err := client.Do(req)
		errors.YLANotifyOnError(err)
		if err != nil {
			YLANewSaveRequestWithAuth("", 400, err.Error(), url, acc.AuthUsername, acc.AuthPassword)
			//spew.Dump(err)
		}
		body, _ := ioutil.ReadAll(resp.Body)
		YLANewSaveRequestWithAuth("", 200, string(body), url, acc.AuthUsername, acc.AuthPassword)
		err = json.Unmarshal(body, &i)
		errors.YLANotifyOnError(err)
		//fmt.Println(i)
		return YLANewGetiKentooCode(card.UUID, acc.AuthUsername, acc.AuthPassword, user)
	} else {
		return 400, "Invalid Business ID"
	}
}

func YLANewGetiKentooCode(carduuid string, AuthUsername string, AuthPassword string, user ylamodels.YLAUser) (int, ylamodels.IkentooQRTokenResponse) {
	var i ylamodels.IkentooQRTokenResponse
	url := "https://pos.ikentoo.com/integration/1/card/" + carduuid + "/qrToken"
	req, err := http.NewRequest("GET", url, nil)
	errors.YLANotifyOnError(err)
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(AuthUsername, AuthPassword)
	client := &http.Client{}
	resp, err := client.Do(req)
	errors.YLANotifyOnError(err)
	if err != nil {
		YLANewSaveRequestWithAuth("", 400, err.Error(), url, AuthUsername, AuthPassword)
	}
	body, _ := ioutil.ReadAll(resp.Body)
	YLANewSaveRequestWithAuth("", 200, string(body), url, AuthUsername, AuthPassword)
	err = json.Unmarshal(body, &i)
	errors.YLANotifyOnError(err)
	i.User = user
	return 200, i
}

func YLANewLookupAddress(addr ylamodels.YLAUserAddress) ([]ylamodels.LookupAddress, error) {
	// 13711667d3fc13
	baseURL, _ := url.Parse("https://eu1.locationiq.com")
	baseURL.Path += "v1/search.php"
	params := url.Values{}

	// Access Key
	params.Add("key", "13711667d3fc13")
	params.Add("street", addr.Address)
	params.Add("city", addr.Town)
	params.Add("postalcode", addr.Postal)
	params.Add("country", "South Africa")
	params.Add("format", "json")
	params.Add("limit", "1")
	baseURL.RawQuery = params.Encode()
	req, _ := http.NewRequest("GET", baseURL.String(), nil)
	res, httperror := http.DefaultClient.Do(req)
	errors.YLASentryCapture(httperror)
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	lookup := []ylamodels.LookupAddress{}
	err := json.Unmarshal(body, &lookup)
	errors.YLASentryCapture(err)
	return lookup, err
}

func YLANewUpdateAddress(address ylamodels.YLAUserAddress) (int, interface{}) {
	var db = database.YLADebuggedConnection()
	db.Save(&address)
	return 200, address
}

func YLANewDeleteAddress(id int) (int, interface{}) {
	address := ylamodels.YLAUserAddress{}
	if id > 0 {
		var db = database.YLADebuggedConnection()
		db.Where("id = ?", id).Find(&address)
		if address.ID > 0 {
			db.Delete(&address)
		}
	}
	return 200, address
}

// YLALambdaUpdateCard... Update Card
func YLALambdaUpdateCard(card ylamodels.YLAUserCard) (int, interface{}) {
	var db = database.YLADebuggedConnection()
	db.Save(&card)
	return 200, card
}

// YLALambdaAddCard... Add New Card
func YLALambdaAddCard(card ylamodels.YLAUserCard) (int, interface{}) {
	var db = database.YLADebuggedConnection()
	db.Create(&card)
	return 200, card
}

// YLALambdaAddCard... Add New Card
func YLALambdaGetCard(id int, db *gorm.DB) ylamodels.YLAUserCard {
	card := ylamodels.YLAUserCard{}
	db.Where("id = ?", id).Find(&card)
	return card
}

// YLALambdaDeleteCard... Delete Card By ID
func YLALambdaDeleteCard(id int) (int, interface{}) {
	var db = database.YLADebuggedConnection()
	card := ylamodels.YLAUserCard{}
	if id > 0 {
		db.Where("id = ?", id).Find(&card)
		if card.ID > 0 {
			//cancel the token
			payfastCancelToken(card, db)
			db.Delete(&card)
		}
	}
	return 200, card
}

func YLALambdaGetUserByUUID(userid int, uuid string) (int, interface{}) {
	var db = database.YLADebuggedConnection()
	data := ylamodels.YLAUser{}
	db.Where("id = ?", userid).First(&data)
	db.Where("user_id = ?", userid).Find(&data.Addresses)
	db.Where("user_id = ?", userid).Find(&data.Cards)
	data.Cards = payfastCheckToken(data.Cards, db)
	database.CloseDatabase(db)
	return 200, data
}

func YLALambdaFindBranchID(branchid int, db *gorm.DB) ylamodels.YLABranch {
	var branch ylamodels.YLABranch
	db.Where("id = ?", branchid).First(&branch)
	return branch
}

func YLALambdaFindBranchByBusiness(branchid string, db *gorm.DB) ylamodels.YLABranch {
	var branch ylamodels.YLABranch
	db.Where("ikentoo_business_id = ?", branchid).First(&branch)
	return branch
}

func YLALambdaFindBranchByLocationID(locationid string, db *gorm.DB) ylamodels.YLABranch {
	var branch ylamodels.YLABranch
	db.Where("ikentoo_id = ?", locationid).First(&branch)
	return branch
}

func YLALambdaITNCreateZapper(itn ylamodels.PaymentITN, orderid int) {
	db := database.YLADebuggedConnection()
	db.Create(&itn)
	//fullorder := ylaactions.GetOrderDetailsFromID(orderid)
	theorder := &ylamodels.YLAOrder{}
	db.Where("id = ?", orderid).First(&theorder)
	if theorder.ID > 0 {
		theorder.PfStatus = itn.PaymentStatus
		db.Save(&theorder)
	}
	if theorder.ID > 0 {
		//d := &ylamodels.YLAOrderInstruction{}
		//d.OrderID = theorder.ExternalReference
		//time.Sleep(10 * time.Second)
		//YLAInjectOrder(*d)
		YLALambdaRetryOrderInjection(theorder.ID)
	}
	database.CloseDatabase(db)
}

func YLALambdaTokenITN(q map[string][]string) {
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
	fmt.Println(itn)
	db := database.YLADebuggedConnection()
	db.Where("pf_merchant_id = ?", itn.MerchantID).First(&branch)
	fmt.Println(branch)
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
		} else if cardid == "CARD" {
			userID := s[1]
			thecardID := s[2]
			card := ylamodels.YLAUserCard{}
			db.Where("id = ?", thecardID).Where("user_id = ?", userID).First(&card)
			card.Token = itn.Token
			db.Save(&card)
		} else {
			userID := s[1]
			card := ylamodels.YLAUserCard{}
			db.Where("id = ?", cardid).Where("user_id = ?", userID).First(&card)
			card.Token = itn.Token
			db.Save(&card)
		}
	}
	database.CloseDatabase(db)
	if theorder.ID > 0 {
		fmt.Println("INJECTING A TOKEN CARD PAYMENT ORDER")
		if theorder.PfStatus == "complete" {
			fmt.Println("INJECTING A COMPLETE CARD PAYMENT ORDER")
			YLALambdaRetryOrderInjection(theorder.ID)
		}
	}
}

func YLALambdaCardITN(q map[string][]string) {
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
	fmt.Println(itn)
	db := database.YLADebuggedConnection()
	db.Where("pf_merchant_id = ?", itn.MerchantID).First(&branch)
	fmt.Println(branch)
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
		} else if cardid == "CARD" {
			userID := s[1]
			thecardID := s[2]
			card := ylamodels.YLAUserCard{}
			db.Where("id = ?", thecardID).Where("user_id = ?", userID).First(&card)
			card.Token = itn.Token
			db.Save(&card)
		} else {
			userID := s[1]
			card := ylamodels.YLAUserCard{}
			db.Where("id = ?", cardid).Where("user_id = ?", userID).First(&card)
			card.Token = itn.Token
			db.Save(&card)
		}
	}
	database.CloseDatabase(db)
	if theorder.ID > 0 {
		fmt.Println("INJECTING A CARD PAYMENT ORDER")
		if theorder.PfStatus == "complete" {
			fmt.Println("INJECTING A COMPLETE CARD PAYMENT ORDER")
			YLALambdaRetryOrderInjection(theorder.ID)
		}
	}
}

func YLALambdaNewAddress(address ylamodels.YLAUserAddress) ylamodels.YLAUserAddress {
	var db = database.YLADebuggedConnection()
	db.Create(&address)
	database.CloseDatabase(db)
	return address
}

func YLALambdaUpdateAddress(address ylamodels.YLAUserAddress) ylamodels.YLAUserAddress {
	db := database.YLADebuggedConnection()
	db.Save(&address)
	database.CloseDatabase(db)
	return address
}

func YLALambdaLookupAddressLocationIQ(addr ylamodels.YLAUserAddress) ([]ylamodels.LookupAddress, error) {
	// 13711667d3fc13
	baseURL, _ := url.Parse("https://eu1.locationiq.com")
	baseURL.Path += "v1/search.php"
	params := url.Values{}
	// Access Key
	params.Add("key", "13711667d3fc13")
	params.Add("street", addr.Address)
	params.Add("city", addr.Town)
	params.Add("postalcode", addr.Postal)
	params.Add("country", "South Africa")
	params.Add("format", "json")
	params.Add("limit", "1")
	baseURL.RawQuery = params.Encode()
	req, er1 := http.NewRequest("GET", baseURL.String(), nil)
	errors.YLASentryCapture(er1)
	res, er2 := http.DefaultClient.Do(req)
	errors.YLASentryCapture(er2)
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	lookup := []ylamodels.LookupAddress{}
	err := json.Unmarshal(body, &lookup)
	errors.YLASentryCapture(err)
	return lookup, err
}

func YLALambdaLoadUserLoyalty(userid int) ylamodels.YLAUserOffers {
	db := database.YLADebuggedConnection()
	user := YLANewFindUserOnID(userid, db)
	useroffer := ylamodels.YLAUserOffers{}
	if user.ID > 0 {
		db.Where("ikentoo_business_id = ?", user.IkentooBusinessID).Where("deleted_at IS NULL").Where("type <> ?", "geo").Find(&useroffer.Offers)
		db.Where("user_id = ?", user.GUID).Order("id desc").Find(&useroffer.Redemptions)
		for i := range useroffer.Offers {
			subQuery := db.Select("photo_id").Where("sku = ?", useroffer.Offers[i].ItemID).Where("ikentoo_business_id = ?", user.IkentooBusinessID).Table("items")
			db.Where("id IN (?)", subQuery).Find(&useroffer.Offers[i].File)
		}
		for j := range useroffer.Redemptions {
			db.Where("id = ?", useroffer.Redemptions[j].GiftID).Where("ikentoo_business_id = ?", user.IkentooBusinessID).First(&useroffer.Redemptions[j].Item)
		}
	}
	useroffer.Stats, _ = YLALambdaGetIkentooSpending(user, db)
	database.CloseDatabase(db)
	return useroffer
}

func YLALambdaLoadUserLoyaltyExtension(userid int) ylamodels.YLAUserOffersExtension {
	db := database.YLADebuggedConnection()
	user := YLANewFindUserOnID(userid, db)
	useroffer := ylamodels.YLAUserOffersExtension{}
	if user.ID > 0 {
		db.Where("ikentoo_business_id = ?", user.IkentooBusinessID).Where("deleted_at IS NULL").Where("type <> ?", "geo").Find(&useroffer.Offers)
		db.Where("user_id = ?", user.GUID).Order("id desc").Find(&useroffer.Redemptions)
		for i := range useroffer.Offers {
			subQuery := db.Select("photo_id").Where("sku = ?", useroffer.Offers[i].ItemID).Where("ikentoo_business_id = ?", user.IkentooBusinessID).Table("items")
			db.Where("id IN (?)", subQuery).Find(&useroffer.Offers[i].File)
		}
		for j := range useroffer.Redemptions {
			db.Where("id = ?", useroffer.Redemptions[j].GiftID).Where("ikentoo_business_id = ?", user.IkentooBusinessID).First(&useroffer.Redemptions[j].Item)
		}
	}
	useroffer.User = user
	useroffer.Stats, _ = YLALambdaGetIkentooSpending(user, db)
	database.CloseDatabase(db)
	return useroffer
}

func YLALambdaGetIkentooSpending(user ylamodels.YLAUser, db *gorm.DB) (ylamodels.YLAPurchaseStats, error) {
	url := "https://integration.ikentoo.com/integration/1/consumers/" + user.IkentooID + "/purchaseStats?businessId=" + user.IkentooBusinessID
	req, err := http.NewRequest("GET", url, nil)
	errors.YLANotifyOnError(err)
	acc := ylamodels.YLAAccount{}
	i := ylamodels.YLAPurchaseStats{}
	db.Where("ikentoo_id = ?", user.IkentooBusinessID).Find(&acc)
	if acc.ID > 0 {
		req.Header.Set("Content-Type", "application/json")
		req.SetBasicAuth(acc.AuthUsername, acc.AuthPassword)
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			errors.YLANotifyOnError(err)
			return i, err
		}
		body, _ := ioutil.ReadAll(resp.Body)
		err = json.Unmarshal(body, &i)
		errors.YLANotifyOnError(err)
	}
	return i, nil
}

func YLALambdaCreateOrder(order ylamodels.YLAOrder, db *gorm.DB) ylamodels.YLAOrder {
	if db == nil {
		db = database.YLADebuggedConnection()
	}
	db.Create(&order)
	if db == nil {
		database.CloseDatabase(db)
	}
	return order
}

func YLALambdaUpdateOrderDetails(data ylamodels.YLAOrderInstruction) ylamodels.YLAOrderDetails {
	db := database.YLADebuggedConnection()
	orderdetails := ylamodels.YLAOrderDetails{}
	deliveryaddress, err := json.Marshal(data.Address)
	errors.YLANotifyOnError(err)
	orderdetails.OrderID = helper.StringToInt(data.OrderID)
	orderdetails.DeliveryAddress = string(deliveryaddress)
	orderdetails.DeliveryType = data.DeliveryType
	orderdetails.OrderNotes = data.Comment
	orderdetails.OrderReference = data.OrderID
	db.Create(&orderdetails)
	db.Model(&ylamodels.YLAOrder{}).Where("id = ?", orderdetails.OrderID).Update("order_status", 3)
	database.CloseDatabase(db)
	return orderdetails
}

func YLALambdaUpdateOnesignal(user ylamodels.YLAOneSingalUser) ylamodels.YLAUser {
	existing := ylamodels.YLAUser{}
	db := database.YLADebuggedConnection()
	if user.UserID > 0 {
		db.Where("id = ?", user.UserID).First(&existing)
		if existing.ID > 0 {
			existing.OsPushToken = user.OsPushToken
			existing.OsUserID = user.OsUserID
			db.Save(&existing)

		}
	}
	return existing
}

func YLALambdaGetUserPastOrders(userId int, ikentooId int) []ylamodels.YLAPastOrder {
	db := database.YLADebuggedConnection()
	var orders []ylamodels.YLAPastOrder
	db.Where("user_id = ?", userId).Order("id desc").Limit(10).Find(&orders)
	for i := range orders {
		subQuery1 := db.Select("item_sku").Where("order_id = ?", orders[i].ID).Where("order_status NOT IN (100)").Where("ikentoo_business_id = ?", ikentooId).Table("order_items")
		db.Where("sku IN (?)", subQuery1).Where("ikentoo_business_id = ?", ikentooId).Find(&orders[i].Items)
	}
	database.CloseDatabase(db)
	return orders
}

func YLALambdaCancelOrder(orderid int) ylamodels.YLAOrder {
	db := database.YLADebuggedConnection()
	var order ylamodels.YLAOrder
	db.Where("id = ?", orderid).Order("id desc").Limit(10).Find(&order)
	if order.ID > 0 {
		order.OrderStatus = 100
		db.Save(&order)
	}
	return order
}

func YLALambdaRetryOrderInjection(orderid int) (int, interface{}) {
	theorder := YLAGetFullOrderFromID(orderid)
	if len(theorder.IkentooStatus) < 2 && len(theorder.IkentooCollectionCode) < 2 {
		if theorder.ID > 0 {
			if theorder.Branch.ID > 0 {
				if theorder.Branch.IkentooVersion == 2 {
					return YLAV1OrderInjectionLambda(theorder)
				} else if theorder.Branch.IkentooVersion == 3 {
					return YLAV3OrderInjectionLambda(theorder)
				}
			}
		}
	}
	return 400, "ORDER FAILED"
}

func YLALambdaOrderInjection(data ylamodels.YLAOrderInstruction) (int, interface{}) {
	theOrder := YLAGetFullOrderFromReference(data.OrderID)
	if theOrder.ID > 0 {
		if theOrder.Branch.ID > 0 {
			if theOrder.Branch.IkentooVersion == 2 {
				return YLAV1OrderInjectionLambda(theOrder)
			} else if theOrder.Branch.IkentooVersion == 3 {
				return YLAV3OrderInjectionLambda(theOrder)
			}
		}
	}
	return 400, nil
}

func YLAV3OrderInjectionLambda(theOrder ylamodels.YLAOrder) (int, interface{}) {
	ikentooOrder := &ylamodels.YLAV3Order{}
	db := database.YLADebuggedConnection()
	db.Model(&ylamodels.YLAOrder{}).Where("id = ?", theOrder.ID).Update("order_status", 56)
	if theOrder.ID > 0 {
		orderDetails := GetOrderDetailsFromID(theOrder.ID)
		ikentooOrder.BusinessLocationID = helper.StringToInt(theOrder.BranchID)
		ikentooOrder.ThirdPartyReference = theOrder.ExternalReference
		ikentooOrder.EndpointID = "YLAV3CALLBACKV2"
		ikentooOrder.AccountProfileCode = "ONLINE_PICKUP"

		for i := range theOrder.OrderItems {
			//check if item processing is a STAFF TIP ITEM
			if theOrder.OrderItems[i].ItemSku == theOrder.Branch.TipSKU {
				//THIS IS A TIP ITEM
				ikentooOrder.Payment.TipAmount = theOrder.OrderItems[i].Amount
			} else {
				tempItem := YLAV3OrderItemToIkentooOrderItem(theOrder.OrderItems[i])
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
		fmt.Println(ikorder)

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
			fmt.Println(i)
			return resp.StatusCode, i
		}

		return 200, ikentooOrder
	}
	return 200, ikentooOrder
}

func YLAV1OrderInjectionLambda(theOrder ylamodels.YLAOrder) (int, interface{}) {
	ikentooOrder := &ylamodels.IkentooOrder{}
	db := database.YLADebuggedConnection()
	db.Model(&ylamodels.YLAOrder{}).Where("id = ?", theOrder.ID).Update("order_status", 55)
	if theOrder.ID > 0 {
		orderDetails := GetOrderDetailsFromID(theOrder.ID)
		theAccount := GetAccountFromIkentooID(helper.StringToInt(theOrder.User.IkentooBusinessID))
		ikentooOrder.BusinessLocationID = helper.StringToInt(theOrder.BranchID)
		ikentooOrder.ExternalReference = theOrder.ExternalReference
		ikentooOrder.DeliveryMode = "PICKUP"
		if len(theOrder.DeliveryType) > 0 {
			ikentooOrder.DeliveryMode = theOrder.DeliveryType
		}

		for i := range theOrder.OrderItems {
			tempItem := YLAOrderItemToIkentooOrderItem(theOrder.OrderItems[i])
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
		} else {
			ikentooOrder.PaymentAmount = 0.0
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

func YLALambdaSaveV1Callback(callback ylamodels.YLAOrderCallback) {
	db := database.YLADebuggedConnection()
	ikmessage := ylamodels.YLAIkentooMessage{}
	ikmessage.AccountID = callback.AccountID
	ikmessage.AccountIdentifier = callback.AccountIdentifier
	ikmessage.BusinessLocationID = callback.BusinessLocationID
	ikmessage.Message = callback.Message
	ikmessage.OrderID = callback.OrderID
	ikmessage.Origin = callback.Origin
	ikmessage.Status = callback.Status
	db.Create(&ikmessage)
	database.CloseDatabase(db)
}

func YLALambdaSaveV3Callback(callback ylamodels.YLAV3OrderCallback) {
	db := database.YLADebuggedConnection()
	ikmessage := ylamodels.YLAIkentooMessage{}
	ikmessage.AccountID = callback.AccountIdentifier
	ikmessage.AccountIdentifier = callback.AccountIdentifier
	ikmessage.BusinessLocationID = callback.AccountIdentifier
	ikmessage.Message = callback.Reason
	ikmessage.OrderID = callback.OrderID
	ikmessage.Origin = "ONLINE"
	ikmessage.Status = callback.Status
	db.Create(&ikmessage)
	database.CloseDatabase(db)
}

func YLALambdaHandleV3OneSignal(callback ylamodels.YLAV3OrderCallback) {
	db := database.YLADebuggedConnection()
	order := YLAGetFullOrderFromReference(callback.OrderID)
	callb := ylamodels.YLAOrderCallback{}
	callb.AccountID = callback.AccountIdentifier
	callb.AccountIdentifier = callback.AccountIdentifier
	callb.Message = callback.Reason
	callb.OrderID = callback.OrderID
	callb.Origin = "ONLINE"
	callb.Status = callback.Status
	callb.BusinessLocationID = order.BranchID
	database.CloseDatabase(db)
	YLAHandleOneSignal(callb)
}

func ParseRTCallback(body []byte) {
	fmt.Println("ParseRTCallback")
	callback := ylamodels.RealTimeCallBack{}
	err := json.Unmarshal(body, &callback)
	errors.YLASentryCapture(err)
	fmt.Println(err)
	fmt.Println(callback)

	if callback.NotificationType == "UPDATE" {
		CheckCallBelongsOrder(callback)
	} else {
		fmt.Println("Not an UPDATE")
	}
}

func RTCUpdateOrderCollectionCode(orderid int, collectioncode string, db *gorm.DB) {
	db.Model(&ylamodels.YLAOrder{}).Where("id = ?", orderid).Update("ikentoo_collection_code", collectioncode)
}
func RTCUpdateOrderStatuses(orderid int, orderstatus string, db *gorm.DB) {
	db.Model(&ylamodels.YLAOrder{}).Where("id = ?", orderid).Update("ikentoo_status", orderstatus)
	db.Model(&ylamodels.YLAOrder{}).Where("id = ?", orderid).Update("push_notification", orderstatus)
	db.Model(&ylamodels.YLAOrder{}).Where("id = ?", orderid).Update("order_status", 60)
}

func RTCallBacktoV1Callback(fullOrder ylamodels.YLAOrder, cb ylamodels.RealTimeCallBack, DeliveryState string, CollectionCode string) {
	db := database.YLADebuggedConnection()
	callb := ylamodels.YLAOrderCallback{}
	callb.AccountID = cb.Account.AccountNumber
	callb.AccountIdentifier = cb.Account.AccountNumber
	callb.Message = DeliveryState
	callb.OrderID = fullOrder.ExternalReference
	callb.Origin = "RTC"
	callb.Status = DeliveryState
	callb.BusinessLocationID = fullOrder.BranchID

	ikmessage := ylamodels.YLAIkentooMessage{}
	ikmessage.AccountID = cb.Account.AccountNumber
	ikmessage.AccountIdentifier = cb.Account.AccountNumber
	ikmessage.BusinessLocationID = fullOrder.BranchID
	ikmessage.Message = DeliveryState
	ikmessage.OrderID = fullOrder.ExternalReference
	ikmessage.Origin = "RTC"
	ikmessage.Status = DeliveryState
	db.Create(&ikmessage)

	database.CloseDatabase(db)
	YLAHandleOneSignal(callb)
}

func CheckCallBelongsOrder(cb ylamodels.RealTimeCallBack) {
	db := database.YLADebuggedConnection()
	branch := &ylamodels.YLABranch{}
	db.Where("ikentoo_id = ?", cb.BusinessLocationId).Find(&branch)
	if branch.ID > 0 {
		fmt.Println("YLA Branch Callback")
		if len(cb.Account.DeliveryInformation.CollectionCode) > 0 {
			fmt.Println("YLA Collection Code")
			if len(cb.Account.ExternalReferences) > 0 {
				fmt.Println("YLA External Reference")
				refSlice := strings.Split(cb.Account.ExternalReferences[0].Reference, "-")
				if len(refSlice) > 0 {
					fmt.Println("YLA CHECK External Reference")
					fullOrder := YLAGetFullOrderFromReference(refSlice[2])
					if fullOrder.ID > 0 {
						if len(fullOrder.IkentooCollectionCode) < 1 {
							RTCUpdateOrderCollectionCode(fullOrder.ID, cb.Account.DeliveryInformation.CollectionCode, db)
							fullOrder.IkentooCollectionCode = cb.Account.DeliveryInformation.CollectionCode
						}
						fmt.Println(cb.Account.DeliveryInformation.DeliveryState)
						fmt.Println(cb.Account.DeliveryInformation.CollectionCode)
						fmt.Println(fullOrder.ID)
						RTCUpdateOrderStatuses(fullOrder.ID, strings.ToLower(cb.Account.DeliveryInformation.DeliveryState), db)
						RTCallBacktoV1Callback(fullOrder, cb, cb.Account.DeliveryInformation.DeliveryState, cb.Account.DeliveryInformation.CollectionCode)
					}
				}
			} else {
				fmt.Println("YLA NO External Reference")
				fullOrder := YLAGetFullOrderFromCollectionCodeAndBranch(cb.Account.DeliveryInformation.CollectionCode, branch.IkentooID)
				if fullOrder.ID > 0 {
					fmt.Println(cb.Account.DeliveryInformation.DeliveryState)
					fmt.Println(cb.Account.DeliveryInformation.CollectionCode)
					fmt.Println(fullOrder.ID)
					RTCUpdateOrderStatuses(fullOrder.ID, strings.ToLower(cb.Account.DeliveryInformation.DeliveryState), db)
					RTCallBacktoV1Callback(fullOrder, cb, cb.Account.DeliveryInformation.DeliveryState, cb.Account.DeliveryInformation.CollectionCode)
				}
			}
		}
	}
	database.CloseDatabase(db)
}

func YLAWebExtFindUser(email string, ikentooid string, locationid string) (int, interface{}) {
	db := database.YLADebuggedConnection()
	user := YLANewFindUserOnEmail(helper.StringToInt(ikentooid), email, db)
	if user.ID > 0 {
		loyalty := YLALambdaLoadUserLoyaltyExtension(user.ID)
		loyalty.Branch = YLALambdaFindBranchByLocationID(locationid, db)
		fmt.Println(locationid)
		fmt.Println(loyalty.Branch)
		return 200, loyalty
	} else {
		return 400, "User Not Found"
	}
}

func YLALoyaltyFindRedemption(guid string, offerid int, ikentoo_uuid string, ikentoo_luuid string, db *gorm.DB) ylamodels.YLAGiftRedemption {
	redemption := ylamodels.YLAGiftRedemption{}
	db.Where("gift_id = ?", offerid).Where("user_id = ?", guid).Where("ikentoo_uuid = ?", ikentoo_uuid).Where("ikentoo_luuid = ?", ikentoo_luuid).First(&redemption)
	return redemption
}

func YLALoyaltyCreateRedemption(guid string, offerid int, ikentoo_uuid string, ikentoo_luuid string, db *gorm.DB) ylamodels.YLAGiftRedemption {
	redemption := ylamodels.YLAGiftRedemption{}
	redemption.GiftID = offerid
	redemption.IkentooLuuid = ikentoo_luuid
	redemption.IkentooUUID = ikentoo_uuid
	redemption.UserID = guid
	db.Create(&redemption)
	return redemption
}

func YLAGetOfferByID(offerid int, db *gorm.DB) ylamodels.YLAOffer {
	offer := ylamodels.YLAOffer{}
	db.Where("id = ?", offerid).First(&offer)
	return offer
}

func YLALoyaltyClaimItem(userid int, offerid int, ikentoo_uuid string, ikentoo_luuid string) (int, interface{}) {
	db := database.YLADebuggedConnection()
	user := YLANewFindUserOnID(userid, db)
	if user.ID > 0 {
		r := YLALoyaltyFindRedemption(user.GUID, offerid, ikentoo_uuid, ikentoo_luuid, db)
		if r.ID > 0 {
			return 400, "You have already claimed this reward on this bill"
		}
		claimed := YLALoyaltyCreateRedemption(user.GUID, offerid, ikentoo_uuid, ikentoo_luuid, db)
		if claimed.ID > 0 {
			offer := YLAGetOfferByID(offerid, db)
			if offer.ID > 0 {
				YLALoyaltyNotifify(offer, user.ID)
			}
			return 200, claimed
		} else {
			return 400, "Could not claim this reward"
		}
	} else {
		return 400, "Could not find user"
	}
}

func YLAOrderCheckItems(orderid int) (int, interface{}) {
	db := database.YLADebuggedConnection()
	order := YLAGetFullOrderFromID(orderid)
	if order.ID > 0 {
		var found = false
		url := "https://api.ikentoo.com/items/v1/items/?businessLocationId=" + order.BranchID
		var bearer = "Bearer 9934d56e-a393-487f-8a71-e9fbfc1bebbc"
		for i := range order.OrderItems {
			//fmt.Println(url + "&sku=" + order.OrderItems[i].ItemSku)
			req, err := http.NewRequest("GET", url+"&sku="+order.OrderItems[i].ItemSku, nil)
			errors.YLASentryCapture(err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Add("Authorization", bearer)
			client := &http.Client{}
			resp, err := client.Do(req)
			errors.YLASentryCapture(err)
			body, _ := ioutil.ReadAll(resp.Body)
			//fmt.Println(string(body))
			results := ylamodels.IkentooItemAPIResult{}
			err = json.Unmarshal(body, &results)

			for j := range results.Results {
				if results.Results[j].SKU == order.OrderItems[i].ItemSku {
					//fmt.Println(results.Results[j])
					found = true
				}
			}

			if !found {
				return 400, "One of the Items in your basket is invalid , please cancel this order and place a new one or contact the branch."
			}
		}
	}
	database.CloseDatabase(db)
	return 200, orderid
}

func YLAClaimVoucher(input ylamodels.POSVoucherInput) (int, interface{}) {
	//step 1 create offer on the fly
	offer := ylamodels.YLAOffer{}
	offer.Description = &input.Description
	offer.IkentooBusinessID = input.IkentooBusinessID
	offer.IkentooID = &input.IkentooId
	offer.ItemID = 1010101
	offer.Price = float64(input.Price)
	offer.Title = input.Title
	offer.Type = "USERCLAIM"
	db := database.YLADebuggedConnection()
	db.Create(&offer)
	user := YLAGetUserByID(input.Radius, db)

	if user.ID > 0 && offer.ID > 0 {
		redemption := ylamodels.YLAGiftRedemption{}
		redemption.GiftID = offer.ID
		redemption.IkentooLuuid = "VOUCHER"
		redemption.IkentooUUID = "VOUCHER"
		redemption.UserID = user.GUID
		db.Create(&redemption)
		database.CloseDatabase(db)
		YLALoyaltyNotifify(offer, user.ID)
		return 200, offer
	} else {
		database.CloseDatabase(db)
		return 400, "Could Not Save Your Voucher"
	}
}
