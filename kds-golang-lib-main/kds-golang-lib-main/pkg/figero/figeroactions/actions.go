package figeroactions

import (
	"errors"
	"time"

	"github.com/reverent911/kds-golang-lib/pkg/figero/figeromodels"
	"github.com/reverent911/kds-golang-lib/pkg/generic/database"
	"github.com/reverent911/kds-golang-lib/pkg/generic/helper"
)

func FIGEROGetUserByEMail(email string) figeromodels.FIGEROUser {
	db := database.FIGEROConnection()
	user := figeromodels.FIGEROUser{}
	db.Where("email = ?", email).First(&user)
	return user
}

func FIGEROLoginUser(username string, password string) (figeromodels.FIGEROUser, error) {
	user := FIGEROGetUserByEMail(username)
	if !(user.ID > 0) {
		return figeromodels.FIGEROUser{}, errors.New("user not found")
	}
	valid := helper.AuthCheckPassword(user.Password, password)
	if !valid {
		return figeromodels.FIGEROUser{}, errors.New("invalid password")
	}
	return user, nil
}

func FigeroUserLocations(userid int) []figeromodels.FIGEROLocation {
	db := database.FIGEROConnection()
	locations := []figeromodels.FIGEROLocation{}
	db.Where("user_id = ?", userid).Find(&locations)
	return locations
}

func FigeroGetBranchDailyThisMonth(branchid int, year int, month int) []figeromodels.FIGEROLocationDataDaily {
	db := database.FIGEROConnection()
	entries := []figeromodels.FIGEROLocationDataDaily{}
	db.Where("location_id = ?", branchid).Where("YEAR(date) = ?", year).Where("MONTH(date) = ?", month).Find(&entries)
	return entries
}

func FigeroGetBranchMonthlyDataAll(branchid int) []figeromodels.FIGEROLocationMonthly {
	db := database.FIGEROConnection()
	entries := []figeromodels.FIGEROLocationMonthly{}
	db.Where("location_id = ?", branchid).Find(&entries)
	return entries
}

func FigeroGetBranchMonthlyData(branchid int, current_year int, current_month int) []figeromodels.FIGEROLocationMonthly {
	db := database.FIGEROConnection()
	monthlyEntries := []figeromodels.FIGEROLocationMonthly{}
	branch := figeromodels.FIGEROLocation{}
	db.Where("id = ?", branchid).First(&branch)
	if branch.ID > 0 && helper.StringToInt(branch.FinYearEndMonth) > -1 {
		if current_month > helper.StringToInt(branch.FinYearEndMonth) {
			//get all monthly data from year start to year end
			db.Where(
				db.Where("YEAR(date) = ?", current_year).Where("MONTH(date) > ?", helper.StringToInt(branch.FinYearEndMonth)),
			).Or(
				db.Where("YEAR(date) = ?", current_year+1).Where("MONTH(date) <= ?", helper.StringToInt(branch.FinYearEndMonth)),
			).Where("location_id = ?", branchid).Find(&monthlyEntries)
		} else {
			db.Where(
				db.Where("YEAR(date) = ?", current_year).Where("MONTH(date) <= ?", helper.StringToInt(branch.FinYearEndMonth)),
			).Or(
				db.Where("YEAR(date) = ?", current_year-1).Where("MONTH(date) > ?", helper.StringToInt(branch.FinYearEndMonth)),
			).Where("location_id = ?", branchid).Find(&monthlyEntries)
		}
	}
	return monthlyEntries
}

func FigeroGetBranchYearlyStats(branchid int, year int) figeromodels.FIGEROYearlyStatsFromDaily {
	db := database.FIGEROConnection()
	branch := figeromodels.FIGEROLocation{}
	entry := figeromodels.FIGEROYearlyStatsFromDaily{}
	t := time.Now()
	db.Where("id = ?", branchid).First(&branch)
	if branch.ID > 0 && helper.StringToInt(branch.FinYearEndMonth) > -1 {
		if t.Year() == year {
			//Current Year
			if int(t.Month()) > helper.StringToInt(branch.FinYearEndMonth) {
				//Past Year End , Need to Look Into Year + 1
				db.Raw("SELECT SUM(income) as total_income, SUM(gross) as total_gross, SUM(net) as total_net FROM location_datas WHERE location_id = ? AND ((YEAR(date) = ? AND MONTH(date) > ?) OR (YEAR(date) = ? AND MONTH(date) <= ?)) LIMIT 1", branchid, year, helper.StringToInt(branch.FinYearEndMonth), year+1, helper.StringToInt(branch.FinYearEndMonth)).Scan(&entry)
			} else {
				//Before Year End , We need to look into last year as well
				db.Raw("SELECT SUM(income) as total_income, SUM(gross) as total_gross, SUM(net) as total_net FROM location_datas WHERE location_id = ? AND ((YEAR(date) = ? AND MONTH(date) <= ?) OR (YEAR(date) = ? AND MONTH(date) > ?)) LIMIT 1", branchid, year, helper.StringToInt(branch.FinYearEndMonth), year-1, helper.StringToInt(branch.FinYearEndMonth)).Scan(&entry)
			}
		} else {
			//@TODO : Request is not for current year
		}
		//db.Raw("SELECT SUM(income) as total_income, SUM(gross) as total_gross, SUM(net) as total_net FROM location_datas WHERE location_id = ? AND YEAR(date) = ? AND MONTH(date) > ? LIMIT 1", branchid, year, helper.StringToInt(branch.FinYearEndMonth)).Scan(&entry)
	}
	return entry
}
