package ylaactions

import (
	"testing"
)

func TestRunTestHanno(t *testing.T) {
	YLAOrderCheckItems(58730)
}

/*
func TestGetYLAConsumerSpend(t *testing.T) {
	abc, result := GetYLAConsumerSpend(9521, 7251)
	str, _ := json.Marshal(result)
	fmt.Println(string(str))
	if abc < 1 {
		t.Fatal("No Data Found")
	}
}

func TestYLANewProcessFlowRegisterUser(t *testing.T) {
	user := &ylamodels.IkentooConsumer{}
	user.Email = "hanno.bean.1@gmail.com"
	user.FirstName = "Hanno"
	user.LastName = "Bean"
	user.PhoneNumber1 = "0836101665"
	user.IkentooBusinessID = 3666
	YLANewProcessFlowRegisterUser(*user)
}
*/
