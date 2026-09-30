package figeroactions

import (
	"fmt"
	"testing"
)

func TestFigeroGetBranchDailyThisMonth(t *testing.T) {
	perms := FigeroGetBranchMonthlyDataAll(214)
	fmt.Println(perms)
}
