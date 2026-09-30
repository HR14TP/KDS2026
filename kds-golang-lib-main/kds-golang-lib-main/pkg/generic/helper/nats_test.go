package helper

import "testing"

func TestTestNats(t *testing.T) {
	TestNats()
	a := 1
	if a < 1 {
		t.Fatal("No Accounts Found")
	}
}
