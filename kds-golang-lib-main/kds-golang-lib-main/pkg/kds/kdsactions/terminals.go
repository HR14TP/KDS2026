package kdsactions

import (
	"github.com/reverent911/kds-golang-lib/pkg/generic/database"
	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
	"gorm.io/gorm"
)

func KDSGetTerminalFromSerial(serial string) kdsmodels.KDSTerminal {
	db := database.KDSConnection()
	kdsterminal := kdsmodels.KDSTerminal{}
	db.Where("connection_code = ?", serial).First(&kdsterminal)
	return kdsterminal
}

func KDSGetAccountsForUser(userid int) []kdsmodels.KDSUserAccounts {
	db := database.KDSConnection()
	kdsaccounts := []kdsmodels.KDSUserAccounts{}
	db.Where("user_id = ?", userid).Find(&kdsaccounts)

	return kdsaccounts
}
func KDSGetAccountFromID(accountid int, db gorm.DB) kdsmodels.KDSAccount {
	kdsaccount := kdsmodels.KDSAccount{}
	db.Where("id = ?", accountid).First(&kdsaccount)
	return kdsaccount
}
func KDSUpdateAccountsForUser(userid int, accounts []int) int {
	db := database.KDSConnection()
	kdsaccounts := []kdsmodels.KDSUserAccounts{}
	db.Where("user_id = ?", userid).Delete(&kdsaccounts)
	for i := range accounts {
		kaccount := kdsmodels.KDSUserAccounts{}
		kaccount.AccountID = accounts[i]
		kaccount.UserID = userid
		db.Create(&kaccount)
	}
	return userid
}

// Update the terminal with serial to bind to the new HWID
func KDSUpdateTerminalRegistration(serial string, hwid string) kdsmodels.KDSTerminal {
	//fmt.Println("UPDATING HWID")
	db := database.KDSConnection()
	kdsterminal := kdsmodels.KDSTerminal{}
	db.Where("connection_code = ?", serial).First(&kdsterminal)
	if kdsterminal.ID > 0 {
		kdsterminal.HWID = hwid
		db.Save(&kdsterminal)
	}
	return kdsterminal
}
