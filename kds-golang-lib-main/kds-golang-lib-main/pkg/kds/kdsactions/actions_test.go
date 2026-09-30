package kdsactions

import (
	"fmt"
	"testing"
)

/*func TestKDSLoadApiData(t *testing.T) {
	user := KDSGetUserByID(27)
	fmt.Println(user)
	result := KDSLoadApiData(user, "today")
	str, _ := json.Marshal(result)
	fmt.Println(string(str))
	if len(result) < 1 {
		t.Fatal("No Data Found")
	}
}*/

/*
func TestGenWaiterSummaryReport(t *testing.T) {
	userid := 12
	result := GenerateDetailReportWithItems(userid, "2022/01/30 13:00:00", "2022/01/30 14:00:00", "XLSX")
	fmt.Println(result)
	if len(result) < 1 {
		t.Fatal("No Accounts Found")
	}
}
*/

func TestKDSPermissions(t *testing.T) {
	perms := GenerateBillingReportXLS()
	fmt.Println(perms)
}
