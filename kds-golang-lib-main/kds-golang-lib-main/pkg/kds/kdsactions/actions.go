package kdsactions

import (
	"errors"

	"github.com/reverent911/kds-golang-lib/pkg/generic/database"
	"github.com/reverent911/kds-golang-lib/pkg/generic/helper"
	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
	"gorm.io/gorm"
)

func getApiDataUserAccountsFromUserID(userid int, db gorm.DB) []kdsmodels.KDSApiDataAccount {
	useraccounts := []kdsmodels.KDSApiDataAccount{}
	db.Where("user_id = ?", userid).Find(&useraccounts)
	return useraccounts
}
func getApiDataTerminalsFromAccountID(accountid int, db gorm.DB) []kdsmodels.KDSApiDataTerminal {
	terminals := []kdsmodels.KDSApiDataTerminal{}
	db.Where("account_id = ?", accountid).Find(&terminals)
	return terminals
}
func getApiDataDailyOrders(terminalid int, db gorm.DB) []kdsmodels.KDSApiDataOrders {
	//WHERE  CAST(Submission_date AS DATE) = CAST( curdate() AS DATE)
	var orders []kdsmodels.KDSApiDataOrders
	db.Where("terminal_id = ?", terminalid).Where("CAST(created_at_on_kds AS DATE) = CAST(CURDATE() AS DATE)").Find(&orders)
	return orders
}
func getApiDataYesterDailyOrders(terminalid int, db gorm.DB) []kdsmodels.KDSApiDataOrders {
	//WHERE  CAST(Submission_date AS DATE) = CAST( curdate() AS DATE)
	var orders []kdsmodels.KDSApiDataOrders
	db.Where("terminal_id = ?", terminalid).Where("CAST(created_at_on_kds AS DATE) = CAST((CURDATE() - INTERVAL 1 day) AS DATE)").Find(&orders)
	return orders
}
func getApiDataOrderItems(terminalid int, orderid int, db gorm.DB) []kdsmodels.KDSApiDataItems {
	var items []kdsmodels.KDSApiDataItems
	db.Where("terminal_id = ?", terminalid).Where("kds_order_id = ?", orderid).Find(&items)
	return items
}
func getApiDataOrderSubItems(terminalid int, mainitemid int, db gorm.DB) []kdsmodels.KDSApiDataSubitems {
	var subitems []kdsmodels.KDSApiDataSubitems
	db.Where("terminal_id = ?", terminalid).Where("kds_item_id = ?", mainitemid).Find(&subitems)
	return subitems
}

func getAllUsersBillingAccounts(db *gorm.DB) []kdsmodels.KDSUser {
	var kdsusers []kdsmodels.KDSUser
	db.Where("user_type_id IN (1,2)").Find(&kdsusers)
	return kdsusers
}

func KDSGetAllUserAccounts(userid int, db gorm.DB) []kdsmodels.KDSApiDataAccount {
	var kdsuseraccounts []kdsmodels.KDSUserAccounts
	db.Where("user_id = ?", userid).Find(&kdsuseraccounts)
	var accountids []int
	for i := range kdsuseraccounts {
		accountids = append(accountids, kdsuseraccounts[i].AccountID)
	}
	var accounts []kdsmodels.KDSApiDataAccount
	if len(accountids) > 0 {
		db.Where("id IN ?", accountids).Find(&accounts)
	}
	return accounts
}
func KDSGetConfigS3() string {
	return helper.DownloadFileAsString("https://kdsdailybackup.s3-eu-west-1.amazonaws.com/releases/config/template.json")
}

func getUserAccountsFromUserID(userid int, db gorm.DB) []kdsmodels.KDSUserAccounts {
	useraccounts := []kdsmodels.KDSUserAccounts{}
	db.Where("user_id = ?", userid).Find(&useraccounts)
	return useraccounts
}

func getTerminalsFromAccountID(accountid int, db gorm.DB) []kdsmodels.KDSTerminal {
	terminals := []kdsmodels.KDSTerminal{}
	db.Where("account_id = ?", accountid).Find(&terminals)
	return terminals
}

func getAccountFromAccountID(accountid int, db gorm.DB) kdsmodels.KDSAccount {
	account := kdsmodels.KDSAccount{}
	db.Where("id = ?", accountid).First(&account)
	return account
}

func getAccountsFromUserID(userid int, db gorm.DB) []kdsmodels.KDSAccount {
	account := []kdsmodels.KDSAccount{}
	db.Where("user_id = ?", userid).Find(&account)
	return account
}

func getLastOrderSynced(terminalid int, db *gorm.DB) kdsmodels.KDSOrders {
	order := kdsmodels.KDSOrders{}
	db.Where("terminal_id = ?", terminalid).Order("id desc").First(&order)
	return order
}

func getTerminalConfigFromTerminalID(terminalid int, db gorm.DB) kdsmodels.KDSTerminalConfig {
	config := kdsmodels.KDSTerminalConfig{}
	db.Where("terminal_id = ?", terminalid).First(&config)
	return config
}

func getTerminalConfigFromSerialKey(terminalid int, db gorm.DB) kdsmodels.KDSTerminalConfig {
	config := kdsmodels.KDSTerminalConfig{}
	db.Where("terminal_id = ?", terminalid).First(&config)
	return config
}

func GetUserReports(userid int) []kdsmodels.KDSReports {
	db := database.KDSConnection()
	kdsreports := []kdsmodels.KDSReports{}
	db.Where("user_id = ?", userid).Find(&kdsreports)
	return kdsreports
}

func DeleteUserReport(userid int, reportid int) []kdsmodels.KDSReports {
	db := database.KDSConnection()
	kdsreports := kdsmodels.KDSReports{}
	db.Where("user_id = ?", userid).Where("id = ?", reportid).Delete(&kdsreports)
	return GetUserReports(userid)
}

func KDSLoginUser(username string, password string) (kdsmodels.KDSUser, error) {
	kdsuser := KDSGetUserByEMail(username)
	if !(kdsuser.ID > 0) {
		return kdsmodels.KDSUser{}, errors.New("user not found")
	}
	valid := helper.AuthCheckPassword(kdsuser.Password, password)
	if !valid {
		return kdsmodels.KDSUser{}, errors.New("invalid password")
	}
	return kdsuser, nil
}

func KDSGetUserByEMail(email string) kdsmodels.KDSUser {
	db := database.KDSConnection()
	kdsuser := kdsmodels.KDSUser{}
	db.Where("email = ?", email).First(&kdsuser)
	return kdsuser
}
func KDSGetUserByID(id int) kdsmodels.KDSUser {
	db := database.KDSConnection()
	kdsuser := kdsmodels.KDSUser{}
	db.Where("id = ?", id).First(&kdsuser)
	return kdsuser
}

func KDSLoadApiData(user kdsmodels.KDSUser, timeframe string) []kdsmodels.KDSApiDataAccount {
	db := database.KDSConnection()
	accounts := KDSGetAllUserAccounts(user.ID, *db)
	for i := range accounts {
		accounts[i].Terminals = getApiDataTerminalsFromAccountID(accounts[i].ID, *db)
		for j := range accounts[i].Terminals {
			if timeframe == "yesterday" {
				accounts[i].Terminals[j].Orders = getApiDataYesterDailyOrders(accounts[i].Terminals[j].ID, *db)
			} else {
				accounts[i].Terminals[j].Orders = getApiDataDailyOrders(accounts[i].Terminals[j].ID, *db)
			}
			for k := range accounts[i].Terminals[j].Orders {
				accounts[i].Terminals[j].Orders[k].Items = getApiDataOrderItems(accounts[i].Terminals[j].ID, accounts[i].Terminals[j].Orders[k].ID, *db)
				for l := range accounts[i].Terminals[j].Orders[k].Items {
					accounts[i].Terminals[j].Orders[k].Items[l].SubItems = getApiDataOrderSubItems(accounts[i].Terminals[j].ID, accounts[i].Terminals[j].Orders[k].Items[l].ID, *db)
				}
			}
		}
	}
	return accounts
}

func KDSApplicationGetConfig(input kdsmodels.KDSVersionCheckInput) kdsmodels.KDSApplicationMySQL {
	application := kdsmodels.KDSApplicationMySQL{}
	db := database.KDSConnection()
	db.Where("name = ?", input.Name).Find(&application)
	return application
}

func KDSApplicationConfigUpdate(input kdsmodels.KDSApplicationMySQL) kdsmodels.KDSApplicationMySQL {
	application := kdsmodels.KDSApplicationMySQL{}
	db := database.KDSConnection()
	db.Where("name = ?", input.Name).Find(&application)
	if application.ID > 0 {
		application.Info = input.Info
		application.Sha1 = input.Sha1
		application.URL = input.URL
		application.Version = input.Version
		application.AppDomain = input.AppDomain
		db.Save(&application)
		return application
	} else {
		db.Create(&input)
		return input
	}
}

func KDSPermissionCreate(input kdsmodels.KDSPermissionsMySQL) kdsmodels.KDSPermissionsMySQL {
	db := database.KDSConnection()
	if input.ID > 0 {
		db.Save(&input)
	} else {
		db.Create(&input)
	}
	return input
}
func KDSCreateUserTypePermission(input kdsmodels.KDSUserTypePermissionsMySQL) kdsmodels.KDSUserTypePermissionsMySQL {
	db := database.KDSConnection()
	if input.ID > 0 {
		db.Save(&input)
	} else {
		db.Create(&input)
	}
	return input
}
func KDSGetUserTypePermissions(usertypeid int) []kdsmodels.KDSPermissionsMySQL {
	db := database.KDSConnection()
	permissions := []kdsmodels.KDSPermissionsMySQL{}
	subQuery1 := db.Model(&kdsmodels.KDSUserTypePermissionsMySQL{}).Select("permission_id").Where("user_type_id = ?", usertypeid)
	db.Where("id in (?)", subQuery1).Find(&permissions)
	return permissions
}
func KDSGetUserPermissions(userid int) []kdsmodels.KDSPermissionsMySQL {
	db := database.KDSConnection()

	subQuery0 := db.Model(&kdsmodels.KDSUser{}).Select("user_type_id").Where("id = ?", userid)
	permissions := []kdsmodels.KDSPermissionsMySQL{}
	subQuery1 := db.Model(&kdsmodels.KDSUserTypePermissionsMySQL{}).Select("permission_id").Where("user_type_id = (?)", subQuery0)
	db.Where("id in (?)", subQuery1).Find(&permissions)
	return permissions
}
func KDSDELETEPermission(id int) interface{} {
	if id > 0 {
		db := database.KDSConnection()
		db.Delete(&kdsmodels.KDSPermissionsMySQL{}, id)
	}
	return "OK"
}
func KDSDELETEUserTypePermissions(id int) interface{} {
	if id > 0 {
		db := database.KDSConnection()
		db.Where("user_type_id = ?", id).Delete(&kdsmodels.KDSUserTypePermissionsMySQL{})
	}
	return "OK"
}
func KDSPermissionsGetAll() []kdsmodels.KDSPermissionsMySQL {
	db := database.KDSConnection()
	permissions := []kdsmodels.KDSPermissionsMySQL{}
	db.Find(&permissions)
	return permissions
}
func KDSUserTypesGetAll() []kdsmodels.KDSUserTypesMySQL {
	db := database.KDSConnection()
	usertypes := []kdsmodels.KDSUserTypesMySQL{}
	db.Find(&usertypes)
	return usertypes
}
func KDSSaveUser(kdsuser kdsmodels.KDSUser) kdsmodels.KDSUser {
	db := database.KDSConnection()
	if kdsuser.ID > 0 {
		db.Save(&kdsuser)
	} else {
		kdsuser.Password = helper.AuthCreatePassword(kdsuser.Password)
		db.Create(&kdsuser)
	}
	return kdsuser
}
func KDSGetUsersForUser(id int) []kdsmodels.KDSUser {
	db := database.KDSConnection()
	users := []kdsmodels.KDSUser{}
	if id > 0 {
		db.Where("user_created_id = ?", id).Find(&users)
	}
	return users
}
func KDSDELETEUser(userid int) kdsmodels.KDSUser {
	db := database.KDSConnection()
	kdsuser := kdsmodels.KDSUser{}
	db.Where("id = ?", userid).Delete(&kdsuser)
	return kdsuser
}

func KDSGetAccountConfigV2(accountID int) kdsmodels.KDSAccountConfigMySQL {
	db := database.KDSConnection()
	config := kdsmodels.KDSAccountConfigMySQL{}
	db.Where("account_id = ?", accountID).First(&config)
	return config
}

func KDSSAVEAccountConfigV2(accountID int, terminalConfig string) kdsmodels.KDSAccountConfigMySQL {
	db := database.KDSConnection()
	config := KDSGetAccountConfigV2(accountID)
	if config.ID > 0 {
		config.Config = terminalConfig
		db.Save(&config)
	} else {
		config.AccountID = accountID
		config.Config = terminalConfig
		db.Create(&config)
	}
	return config
}

// Get the terminal config v2 from the admin portal
func KDSGetTerminalConfigV2(terminalID int) kdsmodels.KDSTerminalConfigV2MySQL {
	db := database.KDSConnection()
	config := kdsmodels.KDSTerminalConfigV2MySQL{}
	db.Where("terminal_id = ?", terminalID).First(&config)
	return config
}

// Save terminal config v2 from the admin portal
func KDSSAVETerminalConfigV2(terminalID int, terminalConfig string) kdsmodels.KDSTerminalConfigV2MySQL {
	db := database.KDSConnection()
	config := KDSGetTerminalConfigV2(terminalID)
	if config.ID > 0 {
		config.Config = terminalConfig
		db.Save(&config)
	} else {
		config.TerminalID = terminalID
		config.Config = terminalConfig
		db.Create(&config)
	}
	return config
}

// Register a v2 terminal via the KDS registration page
func KDSTerminalRegisterV2(hwid string, key string) kdsmodels.KDSTerminal {
	// Step 1 , check if this terminal is already registered
	// terminal := kdsmodels.KDSTerminal{}
	lookup := KDSGetTerminalFromSerial(key)
	if lookup.ID > 0 {
		if lookup.HWID == hwid {
			// terminal is correct and registered to the current hwid
			kdsconfig := KDSGetTerminalConfigV2(lookup.ID)
			lookup.ActiveConfig = kdsconfig.Config
			return lookup
		} else {
			kdsconfig := KDSGetTerminalConfigV2(lookup.ID)
			lookup.ActiveConfig = kdsconfig.Config
			KDSUpdateTerminalRegistration(key, hwid)
			return lookup
		}
	} else {
		return lookup
	}
}

// Lookup a terminal via Key and HWID
func KDSGetTerminalFromSerialandHWID(key string, hwid string) kdsmodels.KDSTerminal {
	db := database.KDSConnection()
	kdsterminal := kdsmodels.KDSTerminal{}
	db.Where("connection_code = ?", key).Where("hwid = ?", hwid).First(&kdsterminal)
	if kdsterminal.ID > 0 {
		if kdsterminal.HWID == hwid {
			kdsconfig := KDSGetTerminalConfigV2(kdsterminal.ID)
			kdsterminal.ActiveConfig = kdsconfig.Config
			return kdsterminal
		}
	}
	return kdsterminal
}

// Update Config from KDS Terminal
func KDSUpdateConfigFromTerminal(input kdsmodels.KDSInputUpdateConfigFromKDS) kdsmodels.KDSTerminal {
	db := database.KDSConnection()
	terminal := KDSGetTerminalFromSerialandHWID(input.Serial, input.HWID)
	if terminal.ID > 0 {
		terminal.ActiveConfig = input.Config
		db.Save(&terminal)
	}
	return terminal
}
