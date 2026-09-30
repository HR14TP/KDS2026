package kdsactions

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/jung-kurt/gofpdf"
	"github.com/reverent911/kds-golang-lib/pkg/generic/awshelper"
	"github.com/reverent911/kds-golang-lib/pkg/generic/database"
	"github.com/reverent911/kds-golang-lib/pkg/generic/helper"
	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type res struct {
	Avg_prep_mins int
	Avg_prep_secs int
	Total_items   int
}
type resSingle struct {
	Single int
}
type resSingleString struct {
	Single string
}

func PDFGENHeader(reportype string, fromdate string, todate string) *gofpdf.Fpdf {
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(0, 10, reportype+" For "+fromdate+" - "+todate, "", 0, "C", false, 0, "")
	pdf.Ln(-1)
	/*
		pdf.SetHeaderFuncMode(func() {
			pdf.SetFont("Arial", "B", 15)
			pdf.CellFormat(0, 10, reportype+" For "+fromdate+" - "+todate, "", 0, "C", false, 0, "")
			pdf.Ln(20)
		}, true)*/
	return pdf
}

func PDFGENTableHeaders(pdf *gofpdf.Fpdf, location string, lineheight float64) *gofpdf.Fpdf {
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 30, "LOCATION : "+location)
	pdf.Ln(30)
	pdf.CellFormat(50, lineheight, "KDS", "1", 0, "", false, 0, "")
	pdf.CellFormat(40, lineheight, "AVG Prep", "1", 0, "C", false, 0, "")
	pdf.CellFormat(40, lineheight, "Items", "1", 0, "C", false, 0, "")
	pdf.CellFormat(30, lineheight, "Orders", "1", 0, "C", false, 0, "")
	pdf.CellFormat(30, lineheight, "Alert Med", "1", 0, "C", false, 0, "")
	pdf.CellFormat(30, lineheight, "Alert High", "1", 0, "C", false, 0, "")
	pdf.CellFormat(30, lineheight, "Voids", "1", 0, "C", false, 0, "")
	pdf.Ln(-1)
	return pdf
}

func PDFGENTableHeadersBilling(pdf *gofpdf.Fpdf, user string, lineheight float64) *gofpdf.Fpdf {
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 30, "USER : "+user)
	pdf.Ln(30)
	pdf.CellFormat(60, lineheight, "ACCOUNT", "1", 0, "", false, 0, "")
	pdf.CellFormat(80, lineheight, "TERMINAL", "1", 0, "C", false, 0, "")
	pdf.CellFormat(50, lineheight, "ACTIVATED", "1", 0, "C", false, 0, "")
	pdf.CellFormat(60, lineheight, "LASTSYNC", "1", 0, "C", false, 0, "")
	pdf.Ln(-1)
	return pdf
}

func PDFGENTableHeadersDetail(pdf *gofpdf.Fpdf, location string, lineheight float64) *gofpdf.Fpdf {
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 30, "LOCATION : "+location)
	pdf.Ln(30)
	pdf.CellFormat(50, lineheight, "Order#", "1", 0, "", false, 0, "")
	pdf.CellFormat(80, lineheight, "Waiter", "1", 0, "C", false, 0, "")
	pdf.CellFormat(50, lineheight, "Table", "1", 0, "C", false, 0, "")
	pdf.CellFormat(40, lineheight, "Profile", "1", 0, "C", false, 0, "")
	pdf.CellFormat(30, lineheight, "Prep Time", "1", 0, "C", false, 0, "")
	pdf.Ln(-1)
	return pdf
}

func PDFGENTableHeadersDetailWithItems(pdf *gofpdf.Fpdf, location string, lineheight float64) *gofpdf.Fpdf {
	//pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 30, "LOCATION : "+location)
	pdf.Ln(30)
	pdf.CellFormat(40, lineheight, "Ref#", "1", 0, "", false, 0, "")
	pdf.CellFormat(40, lineheight, "Time", "1", 0, "", false, 0, "")
	pdf.CellFormat(40, lineheight, "Waiter", "1", 0, "C", false, 0, "")
	pdf.CellFormat(40, lineheight, "Table", "1", 0, "C", false, 0, "")
	pdf.CellFormat(30, lineheight, "Profile", "1", 0, "C", false, 0, "")
	pdf.CellFormat(30, lineheight, "Prep", "1", 0, "C", false, 0, "")
	pdf.CellFormat(60, lineheight, "Items", "1", 0, "C", false, 0, "")
	pdf.Ln(-1)
	return pdf
}

func XLSSheetHeader(xls *excelize.File, sheet string, row int, reportype string, fromdate string, todate string) *excelize.File {
	xls.SetCellValue(sheet, "A"+helper.IntToString(row), reportype+" For "+fromdate+" - "+todate)
	xls.MergeCell(sheet, "A"+helper.IntToString(row), "G"+helper.IntToString(row))
	style, _ := xls.NewStyle(`{"font":{"bold":true}}`)
	xls.SetCellStyle(sheet, "A"+helper.IntToString(row), "G"+helper.IntToString(row), style)
	return xls
}
func XLSGENTableHeadersDetailWithItems(xls *excelize.File, sheet string, row int) *excelize.File {
	//pdf.AddPage()
	xls.SetCellValue(sheet, "A"+helper.IntToString(row), "REF#")
	xls.SetColWidth(sheet, "A", "A", 20)
	xls.SetCellValue(sheet, "B"+helper.IntToString(row), "PREP TIME ORDER")
	xls.SetColWidth(sheet, "B", "B", 20)
	xls.SetCellValue(sheet, "C"+helper.IntToString(row), "TIME")
	xls.SetColWidth(sheet, "C", "C", 20)
	xls.SetCellValue(sheet, "D"+helper.IntToString(row), "WAITER")
	xls.SetColWidth(sheet, "D", "D", 20)
	xls.SetCellValue(sheet, "E"+helper.IntToString(row), "TABLE")
	xls.SetColWidth(sheet, "E", "E", 20)
	xls.SetCellValue(sheet, "F"+helper.IntToString(row), "PROFILE")
	xls.SetColWidth(sheet, "F", "F", 20)
	xls.SetCellValue(sheet, "G"+helper.IntToString(row), "PREP TIME ITEM")
	xls.SetColWidth(sheet, "G", "G", 20)
	xls.SetCellValue(sheet, "H"+helper.IntToString(row), "ITEMS")
	xls.SetColWidth(sheet, "H", "H", 50)

	return xls
}
func XLSGENTableHeadersDetailWithItemsBilling(xls *excelize.File, sheet string, row int) *excelize.File {
	//pdf.AddPage()
	xls.SetCellValue(sheet, "A"+helper.IntToString(row), "USER")
	xls.SetColWidth(sheet, "A", "A", 50)
	xls.SetCellValue(sheet, "B"+helper.IntToString(row), "TERMINAL")
	xls.SetColWidth(sheet, "B", "B", 20)
	xls.SetCellValue(sheet, "C"+helper.IntToString(row), "TERMINAL")
	xls.SetColWidth(sheet, "C", "C", 50)
	xls.SetCellValue(sheet, "D"+helper.IntToString(row), "ACTIVATED")
	xls.SetColWidth(sheet, "D", "D", 20)
	xls.SetCellValue(sheet, "E"+helper.IntToString(row), "LASTSYNC")
	xls.SetColWidth(sheet, "E", "E", 50)
	return xls
}
func XLSEnterTerminalDataBilling(xls *excelize.File, sheet string, row int, terminal kdsmodels.KDSTerminal, account kdsmodels.KDSAccount, activated string, lastsync string, username string) *excelize.File {
	xls.SetCellValue(sheet, "A"+helper.IntToString(row), username)
	xls.SetCellValue(sheet, "B"+helper.IntToString(row), account.Name)
	xls.SetCellValue(sheet, "C"+helper.IntToString(row), terminal.Name)
	xls.SetCellValue(sheet, "D"+helper.IntToString(row), activated)
	xls.SetCellValue(sheet, "E"+helper.IntToString(row), lastsync)
	return xls
}
func XLSEnterTerminal(xls *excelize.File, sheet string, row int, terminal string) *excelize.File {
	xls.SetCellValue(sheet, "A"+helper.IntToString(row), terminal)
	xls.MergeCell(sheet, "A"+helper.IntToString(row), "G"+helper.IntToString(row))
	style, _ := xls.NewStyle(`{"font":{"bold":true}}`)
	xls.SetCellStyle(sheet, "A"+helper.IntToString(row), "G"+helper.IntToString(row), style)
	return xls
}
func XLSEnterTerminalData(xls *excelize.File, sheet string, row int, order kdsmodels.KDSExportOrders, item kdsmodels.KDSExportItemsMySQL, res res) *excelize.File {
	xls.SetCellValue(sheet, "A"+helper.IntToString(row), order.Reference)
	//res := PDFDBGetSummaryPerTerminalAndOrderID(*db, terminal.ID, order.ID)
	xls.SetCellValue(sheet, "B"+helper.IntToString(row), helper.IntToString(res.Avg_prep_mins)+":"+helper.IntToDoubleString(res.Avg_prep_secs))
	xls.SetCellValue(sheet, "C"+helper.IntToString(row), order.Created.Add(-time.Hour*2).Format("2006-01-02 15:04:05"))
	xls.SetCellValue(sheet, "D"+helper.IntToString(row), order.Waiter)
	xls.SetCellValue(sheet, "E"+helper.IntToString(row), order.Table)
	xls.SetCellValue(sheet, "F"+helper.IntToString(row), order.Profile)
	xls.SetCellValue(sheet, "G"+helper.IntToString(row), helper.IntToString(item.PrepMins)+":"+helper.IntToString(item.PrepSecs))
	xls.SetCellValue(sheet, "H"+helper.IntToString(row), item.Name)
	return xls
}

func PDFDBGetSummaryPerTerminal(db gorm.DB, terminid int, fromdate string, todate string) res {
	result := res{}
	db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("AVG(prep_mins) DIV 1 as avg_prep_mins , ROUND(MOD(AVG(prep_mins) , 1) * 60) as avg_prep_secs , SUM(qty) as total_items").Where("terminal_id = ?", terminid).Where("DATE(created) >= ?", fromdate).Where("DATE(created) <= ?", todate).First(&result)
	return result
}

func PDFDBGetSummaryPerTerminalAndOrderID(db gorm.DB, terminid int, orderid int) res {
	result := res{}
	db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("AVG(prep_mins) DIV 1 as avg_prep_mins , ROUND(MOD(AVG(prep_mins) , 1) * 60) as avg_prep_secs , SUM(qty) as total_items").Where("terminal_id = ?", terminid).Where("kds_order_id = ?", orderid).First(&result)
	return result
}

func PDFDBGetOrdersForTime(db gorm.DB, terminid int, fromdate string, todate string) []kdsmodels.KDSExportOrders {
	var orders []kdsmodels.KDSExportOrders
	db.Where("terminal_id = ?", terminid).Where("(created_at_on_kds - interval 2 hour) >= ?", fromdate).Where("(created_at_on_kds - interval 2 hour) <= ?", todate).Find(&orders)
	return orders
}
func PDFDBGetItemsFromOrder(db gorm.DB, terminid int, orderid int) []kdsmodels.KDSExportItemsMySQL {
	var items []kdsmodels.KDSExportItemsMySQL
	//db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("ROUND(cast(AVG(prep_mins) as decimal)) as avg_prep_mins , ROUND(cast(AVG(prep_secs) as decimal)) as avg_prep_secs , SUM(qty) as total_items").Where("terminal_id = ?", terminid).Where("DATE(created) >= ?", fromdate).Where("DATE(created) <= ?", todate).First(&result)
	db.Where("terminal_id = ?", terminid).Where("kds_order_id = ?", orderid).Find(&items)
	return items
}

func GenerateBillingReport() string {
	db := database.KDSConnection()
	users := getAllUsersBillingAccounts(db)
	var lineh float64 = 10
	pdf := PDFGENHeader("Billing Report", "abc", "def")
	for i := range users {
		pdf = PDFGENTableHeadersBilling(pdf, users[i].Email, lineh)
		accounts := getAccountsFromUserID(users[i].ID, *db)
		if len(accounts) > 0 {
			for j := range accounts {
				terminals := getTerminalsFromAccountID(accounts[j].ID, *db)
				for x := range terminals {
					pdf.SetFont("Arial", "", 11)
					pdf.CellFormat(60, lineh, accounts[j].Name, "1", 0, "", false, 0, "")
					pdf.CellFormat(80, lineh, terminals[x].Name, "1", 0, "", false, 0, "")
					if len(terminals[x].HWID) > 5 {
						pdf.CellFormat(50, lineh, "ACTIVATED", "1", 0, "", false, 0, "")
					} else {
						pdf.CellFormat(50, lineh, "DORMANT", "1", 0, "", false, 0, "")
					}
					order := getLastOrderSynced(terminals[x].ID, db)
					if order.ID > 0 {
						pdf.CellFormat(60, lineh, order.Created, "1", 0, "", false, 0, "")
					} else {
						pdf.CellFormat(60, lineh, "N/A", "1", 0, "", false, 0, "")
					}
					pdf.Ln(-1)
				}
			}
		}
	}
	_ = pdf.OutputFileAndClose("/tmp/reportX.pdf")
	return "kdsreports/reportX.pdf"
}

/*
	if len(useraccounts) > 0 {
		// Create a new sheet.
		for i := range useraccounts {
			activerow := 1
			terminals := getTerminalsFromAccountID(useraccounts[i].AccountID, *db)
			account := getAccountFromAccountID(useraccounts[i].AccountID, *db)
			index := f.NewSheet(account.Name)
			f.DeleteSheet("Sheet1")
			f.SetActiveSheet(index)
			f = XLSSheetHeader(f, account.Name, activerow, "Detailed Report With Items", fromdatetime, todatetime)
			activerow += 2
			f = XLSGENTableHeadersDetailWithItems(f, account.Name, activerow)
			activerow += 1
			if len(terminals) > 0 {
				for j := range terminals {
					orders := PDFDBGetOrdersForTime(*db, terminals[j].ID, fromdatetime, todatetime)
					if len(orders) > 0 {
						f = XLSEnterTerminal(f, account.Name, activerow, terminals[j].Name)
						activerow += 1
						for k := range orders {
							items := PDFDBGetItemsFromOrder(*db, terminals[j].ID, orders[k].ID)
							res := PDFDBGetSummaryPerTerminalAndOrderID(*db, terminals[j].ID, orders[k].ID)
							for o := range items {
								f = XLSEnterTerminalData(f, account.Name, activerow, orders[k], items[o], res)
								activerow += 1
							}
						}
					}
				}
			}
*/
func GenerateBillingReportXLS() string {
	db := database.KDSConnection()
	users := getAllUsersBillingAccounts(db)
	f := excelize.NewFile()
	index := f.NewSheet("Billing")
	f.DeleteSheet("Sheet1")
	f.SetActiveSheet(index)
	activerow := 1
	f = XLSSheetHeader(f, "Billing", activerow, "Detailed Report With Items", "fromdatetime", "todatetime")
	activerow += 2
	f = XLSGENTableHeadersDetailWithItemsBilling(f, "Billing", activerow)
	activerow += 2
	for i := range users {
		accounts := getAccountsFromUserID(users[i].ID, *db)
		if len(accounts) > 0 {
			for j := range accounts {
				terminals := getTerminalsFromAccountID(accounts[j].ID, *db)
				for x := range terminals {
					activated := "DORMANT"
					lastsync := "N/A"
					if len(terminals[x].HWID) > 5 {
						activated = "ACTIVATED"
					}
					order := getLastOrderSynced(terminals[x].ID, db)
					if order.ID > 0 {
						lastsync = order.Created
					}
					f = XLSEnterTerminalDataBilling(f, "Billing", activerow, terminals[x], accounts[j], activated, lastsync, users[i].Email)
					activerow += 1
				}
			}
		}
	}
	if err := f.SaveAs("/tmp/report.xlsx"); err != nil {
		fmt.Println(err)
		return "REPORT FAILED"
	}
	return "/tmp/report.xlsx"
}

func GenerateDetailReport(userid int, fromdatetime string, todatetime string) string {
	db := database.KDSConnection()
	useraccounts := getUserAccountsFromUserID(userid, *db)
	var lineh float64 = 10
	pdf := PDFGENHeader("Detailed Report", fromdatetime, todatetime)
	if len(useraccounts) > 0 {
		for i := range useraccounts {
			terminals := getTerminalsFromAccountID(useraccounts[i].AccountID, *db)
			account := getAccountFromAccountID(useraccounts[i].AccountID, *db)
			pdf = PDFGENTableHeadersDetail(pdf, account.Name, lineh)
			pdf.SetFont("Arial", "", 16)
			if len(terminals) > 0 {
				for j := range terminals {
					orders := PDFDBGetOrdersForTime(*db, terminals[j].ID, fromdatetime, todatetime)
					if len(orders) > 0 {
						pdf.SetFont("Arial", "B", 16)
						pdf.Cell(0, 30, "KDS : "+terminals[j].Name)
						pdf.Ln(30)
						pdf.SetFont("Arial", "", 14)
						for k := range orders {
							pdf.CellFormat(50, lineh, orders[k].Reference, "1", 0, "", false, 0, "")
							pdf.CellFormat(80, lineh, orders[k].Waiter, "1", 0, "", false, 0, "")
							pdf.CellFormat(50, lineh, orders[k].Table, "1", 0, "", false, 0, "")
							pdf.CellFormat(40, lineh, orders[k].Profile, "1", 0, "", false, 0, "")

							res := PDFDBGetSummaryPerTerminalAndOrderID(*db, terminals[j].ID, orders[k].ID)
							pdf.CellFormat(30, lineh, helper.IntToString(res.Avg_prep_mins)+":"+helper.IntToDoubleString(res.Avg_prep_secs), "1", 0, "C", false, 0, "")

							//pdf.CellFormat(30, lineh, orders[k].Status, "1", 0, "", false, 0, "")
							pdf.Ln(-1)
						}
					}
				}
			}
		}
		_ = pdf.OutputFileAndClose("/tmp/report.pdf")
		ran := helper.RandStringRunes(10)
		awshelper.KDSUploadReport("/tmp/report.pdf", "kdsdailybackup", "kdsreports/"+ran+".pdf")
		return "kdsreports/" + ran + ".pdf"
	}
	return "DONE"
}

func GenerateDetailReportWithItems(userid int, fromdatetime string, todatetime string, format string) string {
	db := database.KDSConnection()
	useraccounts := getUserAccountsFromUserID(userid, *db)
	if format == "PDF" {
		var lineh float64 = 10
		pdf := PDFGENHeader("Detailed Report", fromdatetime, todatetime)
		if len(useraccounts) > 0 {
			for i := range useraccounts {
				terminals := getTerminalsFromAccountID(useraccounts[i].AccountID, *db)
				account := getAccountFromAccountID(useraccounts[i].AccountID, *db)
				pdf = PDFGENTableHeadersDetailWithItems(pdf, account.Name, lineh)
				pdf.SetFont("Arial", "", 16)
				if len(terminals) > 0 {
					for j := range terminals {
						orders := PDFDBGetOrdersForTime(*db, terminals[j].ID, fromdatetime, todatetime)
						if len(orders) > 0 {
							pdf.SetFont("Arial", "B", 16)
							pdf.Cell(0, 30, "KDS : "+terminals[j].Name)
							pdf.Ln(30)
							pdf.SetFont("Arial", "", 10)
							for k := range orders {
								//get all items for order
								items := PDFDBGetItemsFromOrder(*db, terminals[j].ID, orders[k].ID)
								for o := range items {
									/**
									pdf.CellFormat(40, lineheight, "Ref#", "1", 0, "", false, 0, "")
									pdf.CellFormat(40, lineheight, "Time", "1", 0, "", false, 0, "")
									pdf.CellFormat(40, lineheight, "Waiter", "1", 0, "C", false, 0, "")
									pdf.CellFormat(40, lineheight, "Table", "1", 0, "C", false, 0, "")
									pdf.CellFormat(30, lineheight, "Profile", "1", 0, "C", false, 0, "")
									pdf.CellFormat(30, lineheight, "Prep", "1", 0, "C", false, 0, "")
									pdf.CellFormat(60, lineheight, "Items", "1", 0, "C", false, 0, "")
									*/
									pdf.CellFormat(40, lineh, orders[k].Reference, "1", 0, "", false, 0, "")
									pdf.CellFormat(40, lineh, orders[k].Created.Format("15:04:05"), "1", 0, "", false, 0, "")
									pdf.CellFormat(40, lineh, orders[k].Waiter, "1", 0, "", false, 0, "")
									pdf.CellFormat(40, lineh, orders[k].Table, "1", 0, "", false, 0, "")
									pdf.CellFormat(30, lineh, orders[k].Profile, "1", 0, "", false, 0, "")
									res := PDFDBGetSummaryPerTerminalAndOrderID(*db, terminals[j].ID, orders[k].ID)
									pdf.CellFormat(30, lineh, helper.IntToString(res.Avg_prep_mins)+":"+helper.IntToDoubleString(res.Avg_prep_secs), "1", 0, "C", false, 0, "")
									pdf.CellFormat(60, lineh, items[o].Name, "1", 0, "", false, 0, "")
									pdf.Ln(-1)
								}
							}
						}
					}
				}
			}
			_ = pdf.OutputFileAndClose("/tmp/report.pdf")
			ran := helper.RandStringRunes(10)
			awshelper.KDSUploadReport("/tmp/report.pdf", "kdsdailybackup", "kdsreports/"+ran+".pdf")
			return "kdsreports/" + ran + ".pdf"
		}
	} else if format == "XLSX" {
		f := excelize.NewFile()
		if len(useraccounts) > 0 {
			// Create a new sheet.
			for i := range useraccounts {
				activerow := 1
				terminals := getTerminalsFromAccountID(useraccounts[i].AccountID, *db)
				account := getAccountFromAccountID(useraccounts[i].AccountID, *db)
				index := f.NewSheet(account.Name)
				f.DeleteSheet("Sheet1")
				f.SetActiveSheet(index)
				f = XLSSheetHeader(f, account.Name, activerow, "Detailed Report With Items", fromdatetime, todatetime)
				activerow += 2
				f = XLSGENTableHeadersDetailWithItems(f, account.Name, activerow)
				activerow += 1
				if len(terminals) > 0 {
					for j := range terminals {
						orders := PDFDBGetOrdersForTime(*db, terminals[j].ID, fromdatetime, todatetime)
						if len(orders) > 0 {
							f = XLSEnterTerminal(f, account.Name, activerow, terminals[j].Name)
							activerow += 1
							for k := range orders {
								items := PDFDBGetItemsFromOrder(*db, terminals[j].ID, orders[k].ID)
								res := PDFDBGetSummaryPerTerminalAndOrderID(*db, terminals[j].ID, orders[k].ID)
								for o := range items {
									f = XLSEnterTerminalData(f, account.Name, activerow, orders[k], items[o], res)
									activerow += 1
								}
							}
						}
					}
				}
				/*
					if len(terminals) > 0 {
						for j := range terminals {
							orders := PDFDBGetOrdersForTime(*db, terminals[j].ID, fromdatetime, todatetime)
							if len(orders) > 0 {
								pdf.SetFont("Arial", "B", 16)
								pdf.Cell(0, 30, "KDS : "+terminals[j].Name)
								pdf.Ln(30)
								pdf.SetFont("Arial", "", 10)
								for k := range orders {
									//get all items for order
									items := PDFDBGetItemsFromOrder(*db, terminals[j].ID, orders[k].ID)
									for o := range items {
										pdf.CellFormat(40, lineh, orders[k].Reference, "1", 0, "", false, 0, "")
										pdf.CellFormat(40, lineh, orders[k].Created.Format("15:04:05"), "1", 0, "", false, 0, "")
										pdf.CellFormat(40, lineh, orders[k].Waiter, "1", 0, "", false, 0, "")
										pdf.CellFormat(40, lineh, orders[k].Table, "1", 0, "", false, 0, "")
										pdf.CellFormat(30, lineh, orders[k].Profile, "1", 0, "", false, 0, "")
										res := PDFDBGetSummaryPerTerminalAndOrderID(*db, terminals[j].ID, orders[k].ID)
										pdf.CellFormat(30, lineh, helper.IntToString(res.Avg_prep_mins)+":"+helper.IntToString(res.Avg_prep_secs), "1", 0, "C", false, 0, "")
										pdf.CellFormat(60, lineh, items[o].Name, "1", 0, "", false, 0, "")
										pdf.Ln(-1)
									}
								}
							}
						}
					}*/
			}
		}
		// Save spreadsheet by the given path.
		if err := f.SaveAs("/tmp/report.xlsx"); err != nil {
			fmt.Println(err)
			return "REPORT FAILED"
		} else {
			ran := helper.RandStringRunes(10)
			awshelper.KDSUploadReport("/tmp/report.xlsx", "kdsdailybackup", "kdsreports/"+ran+".xlsx")
			return "kdsreports/" + ran + ".xlsx"
			//return "https://kdsdailybackup.s3.eu-west-1.amazonaws.com/kdsreports/" + ran + ".xlsx"
		}
	}
	return "DONE"
}

func GenerateSummaryReport(userid int, fromdate string, todate string) string {
	db := database.KDSConnection()
	useraccounts := getUserAccountsFromUserID(userid, *db)
	var lineh float64 = 10
	if len(useraccounts) > 0 {
		pdf := PDFGENHeader("Summary Report", fromdate, todate)

		for i := range useraccounts {
			terminals := getTerminalsFromAccountID(useraccounts[i].AccountID, *db)
			account := getAccountFromAccountID(useraccounts[i].AccountID, *db)

			pdf.AddPage()
			pdf.SetFont("Arial", "B", 16)
			pdf.Cell(0, 30, "LOCATION : "+account.Name)
			pdf.Ln(30)
			pdf.CellFormat(50, lineh, "KDS", "1", 0, "", false, 0, "")
			pdf.CellFormat(40, lineh, "AVG Prep", "1", 0, "C", false, 0, "")
			pdf.CellFormat(40, lineh, "Items", "1", 0, "C", false, 0, "")
			pdf.CellFormat(30, lineh, "Orders", "1", 0, "C", false, 0, "")
			pdf.CellFormat(30, lineh, "Alert Med", "1", 0, "C", false, 0, "")
			pdf.CellFormat(30, lineh, "Alert High", "1", 0, "C", false, 0, "")
			pdf.CellFormat(30, lineh, "Voids", "1", 0, "C", false, 0, "")
			pdf.Ln(-1)
			if len(terminals) > 0 {
				for j := range terminals {
					result := res{}
					db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("AVG(prep_mins) DIV 1 as avg_prep_mins , ROUND(MOD(AVG(prep_mins) , 1) * 60) as avg_prep_secs , SUM(qty) as total_items").Where("terminal_id = ?", terminals[j].ID).Where("DATE(created) >= ?", fromdate).Where("DATE(created) <= ?", todate).First(&result)
					if result.Total_items > 0 {
						config := getTerminalConfigFromTerminalID(terminals[j].ID, *db)
						late1 := 2
						late2 := 4
						if config.ID > 0 {
							kdsconfig := kdsmodels.KDSConfig{}
							wrong := json.Unmarshal([]byte(config.Config), &kdsconfig)
							//spew.Dump(wrong)
							//spew.Dump(kdsconfig)
							if wrong == nil {
								if helper.StringToInt(kdsconfig.Colors.Late1time) > 0 {
									late1 = helper.StringToInt(kdsconfig.Colors.Late1time)
								}
								if helper.StringToInt(kdsconfig.Colors.Late2time) > 0 {
									late2 = helper.StringToInt(kdsconfig.Colors.Late2time)
								}
							}
						}

						ressingle := resSingle{}
						pdf.SetFont("Arial", "", 16)
						pdf.CellFormat(50, lineh, terminals[j].Name, "0", 0, "", false, 0, "")
						pdf.CellFormat(40, lineh, helper.IntToString(result.Avg_prep_mins)+":"+helper.IntToDoubleString(result.Avg_prep_secs), "1", 0, "C", false, 0, "")
						pdf.CellFormat(40, lineh, helper.IntToString(result.Total_items), "1", 0, "C", false, 0, "")

						db.Model(&kdsmodels.KDSExportOrders{}).Select("count(1) as single").Where("terminal_id = ?", terminals[j].ID).Where("DATE(created_at_on_kds) >= ?", fromdate).Where("DATE(created_at_on_kds) <= ?", todate).First(&ressingle)
						pdf.CellFormat(30, lineh, helper.IntToString(ressingle.Single), "1", 0, "C", false, 0, "")

						db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("count(1) as single").Where("prep_mins > ?", late1).Where("terminal_id = ?", terminals[j].ID).Where("DATE(created) >= ?", fromdate).Where("DATE(created) <= ?", todate).First(&ressingle)
						pdf.CellFormat(30, lineh, "("+helper.IntToString(late1)+") "+helper.IntToString(ressingle.Single), "1", 0, "C", false, 0, "")

						db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("count(1) as single").Where("prep_mins > ?", late2).Where("terminal_id = ?", terminals[j].ID).Where("DATE(created) >= ?", fromdate).Where("DATE(created) <= ?", todate).First(&ressingle)
						pdf.CellFormat(30, lineh, "("+helper.IntToString(late2)+") "+helper.IntToString(ressingle.Single), "1", 0, "C", false, 0, "")

						db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("count(qty) as single").Where("terminal_id = ?", terminals[j].ID).Where("DATE(created) >= ?", fromdate).Where("DATE(created) <= ?", todate).Where("status = ?", "VOID").First(&ressingle)
						pdf.CellFormat(30, lineh, helper.IntToString(ressingle.Single), "1", 0, "C", false, 0, "")

						pdf.Ln(-1)
					}
				}
			}
		}
		_ = pdf.OutputFileAndClose("/tmp/report.pdf")
		ran := helper.RandStringRunes(10)
		awshelper.KDSUploadReport("/tmp/report.pdf", "kdsdailybackup", "kdsreports/"+ran+".pdf")
		return "kdsreports/" + ran + ".pdf"
	}
	return "DONE"
}

func GenerateSummaryReport2(userid int, fromdate string, todate string) string {
	db := database.KDSConnection()
	useraccounts := getUserAccountsFromUserID(userid, *db)
	var lineh float64 = 10
	if len(useraccounts) > 0 {
		pdf := PDFGENHeader("Summary Report", fromdate, todate)

		for i := range useraccounts {
			terminals := getTerminalsFromAccountID(useraccounts[i].AccountID, *db)
			account := getAccountFromAccountID(useraccounts[i].AccountID, *db)

			pdf.AddPage()
			pdf.SetFont("Arial", "B", 16)
			pdf.Cell(0, 30, "LOCATION : "+account.Name)
			pdf.Ln(30)
			pdf.CellFormat(50, lineh, "KDS", "1", 0, "", false, 0, "")
			pdf.CellFormat(40, lineh, "AVG Prep", "1", 0, "C", false, 0, "")
			pdf.CellFormat(40, lineh, "Items", "1", 0, "C", false, 0, "")
			pdf.CellFormat(30, lineh, "Orders", "1", 0, "C", false, 0, "")
			pdf.CellFormat(30, lineh, "Alert Med", "1", 0, "C", false, 0, "")
			pdf.CellFormat(30, lineh, "Alert High", "1", 0, "C", false, 0, "")
			pdf.CellFormat(30, lineh, "Voids", "1", 0, "C", false, 0, "")
			pdf.Ln(-1)
			if len(terminals) > 0 {
				for j := range terminals {
					result := res{}
					db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("AVG(prep_mins) DIV 1 as avg_prep_mins , ROUND(MOD(AVG(prep_mins) , 1) * 60) as avg_prep_secs , SUM(qty) as total_items").Where("terminal_id = ?", terminals[j].ID).Where("DATE(created) >= ?", fromdate).Where("DATE(created) <= ?", todate).First(&result)
					if result.Total_items > 0 {
						config := getTerminalConfigFromTerminalID(terminals[j].ID, *db)
						late1 := 2
						late2 := 4
						if config.ID > 0 {
							kdsconfig := kdsmodels.KDSConfig{}
							wrong := json.Unmarshal([]byte(config.Config), &kdsconfig)
							//spew.Dump(wrong)
							//spew.Dump(kdsconfig)
							if wrong == nil {
								if helper.StringToInt(kdsconfig.Colors.Late1time) > 0 {
									late1 = helper.StringToInt(kdsconfig.Colors.Late1time)
								}
								if helper.StringToInt(kdsconfig.Colors.Late2time) > 0 {
									late2 = helper.StringToInt(kdsconfig.Colors.Late2time)
								}
							}
						}

						ressingle := resSingle{}
						pdf.SetFont("Arial", "", 16)
						pdf.CellFormat(50, lineh, terminals[j].Name, "0", 0, "", false, 0, "")
						pdf.CellFormat(40, lineh, helper.IntToString(result.Avg_prep_mins)+":"+helper.IntToDoubleString(result.Avg_prep_secs), "1", 0, "C", false, 0, "")
						pdf.CellFormat(40, lineh, helper.IntToString(result.Total_items), "1", 0, "C", false, 0, "")

						db.Model(&kdsmodels.KDSExportOrders{}).Select("count(1) as single").Where("terminal_id = ?", terminals[j].ID).Where("DATE(created_at_on_kds) >= ?", fromdate).Where("DATE(created_at_on_kds) <= ?", todate).First(&ressingle)
						pdf.CellFormat(30, lineh, helper.IntToString(ressingle.Single), "1", 0, "C", false, 0, "")

						db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("count(1) as single").Where("prep_mins > ?", late1).Where("terminal_id = ?", terminals[j].ID).Where("DATE(created) >= ?", fromdate).Where("DATE(created) <= ?", todate).First(&ressingle)
						pdf.CellFormat(30, lineh, "("+helper.IntToString(late1)+") "+helper.IntToString(ressingle.Single), "1", 0, "C", false, 0, "")

						db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("count(1) as single").Where("prep_mins > ?", late2).Where("terminal_id = ?", terminals[j].ID).Where("DATE(created) >= ?", fromdate).Where("DATE(created) <= ?", todate).First(&ressingle)
						pdf.CellFormat(30, lineh, "("+helper.IntToString(late2)+") "+helper.IntToString(ressingle.Single), "1", 0, "C", false, 0, "")

						db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("count(qty) as single").Where("terminal_id = ?", terminals[j].ID).Where("DATE(created) >= ?", fromdate).Where("DATE(created) <= ?", todate).Where("status = ?", "VOID").First(&ressingle)
						pdf.CellFormat(30, lineh, helper.IntToString(ressingle.Single), "1", 0, "C", false, 0, "")

						pdf.Ln(-1)
					}
				}
			}
		}
		_ = pdf.OutputFileAndClose("/tmp/report.pdf")
		ran := helper.RandStringRunes(10)
		awshelper.KDSUploadReport("/tmp/report.pdf", "kdsdailybackup", "kdsreports/"+ran+".pdf")
		return "kdsreports/" + ran + ".pdf"
	}
	return "DONE"
}

func GenerateSummaryReport3(userid int, fromdate string, todate string) string {
	data := DataAccountSummaryReport(userid, fromdate, todate)
	var lineh float64 = 10
	if len(data.Terminals) > 0 {
		pdf := PDFGENHeader("Summary Report", fromdate, todate)
		lastid := 0
		for i := range data.Terminals {
			if lastid != data.Terminals[i].Account.ID {
				pdf.AddPage()
				pdf.SetFont("Arial", "B", 10)
				pdf.Cell(0, 30, "LOCATION : "+data.Terminals[i].Account.Name+" ("+"Summary Report"+" For "+fromdate+" - "+todate+")")
				pdf.Ln(30)
				pdf.CellFormat(35, lineh, "KDS", "1", 0, "", false, 0, "")
				pdf.CellFormat(30, lineh, "AVG Prep", "1", 0, "C", false, 0, "")
				pdf.CellFormat(20, lineh, "Orders", "1", 0, "C", false, 0, "")
				pdf.CellFormat(20, lineh, "Items", "1", 0, "C", false, 0, "")

				pdf.CellFormat(25, lineh, "0-2M", "1", 0, "C", false, 0, "")
				pdf.CellFormat(25, lineh, "0-4M", "1", 0, "C", false, 0, "")
				pdf.CellFormat(25, lineh, "0-6M", "1", 0, "C", false, 0, "")
				pdf.CellFormat(25, lineh, "0-8M", "1", 0, "C", false, 0, "")
				pdf.CellFormat(25, lineh, "0-10M", "1", 0, "C", false, 0, "")
				pdf.CellFormat(25, lineh, "0-15M", "1", 0, "C", false, 0, "")
				pdf.CellFormat(25, lineh, "15M+", "1", 0, "C", false, 0, "")
				pdf.Ln(-1)
			}
			lastid = data.Terminals[i].Account.ID
			if data.Terminals[i].Total_Orders > 0 {
				pdf.SetFont("Arial", "B", 8)
				pdf.CellFormat(35, lineh, data.Terminals[i].Terminal.Name, "0", 0, "", false, 0, "")
				pdf.SetFont("Arial", "", 8)
				pdf.CellFormat(30, lineh, helper.IntToString(data.Terminals[i].Avg_prep_mins)+":"+helper.IntToDoubleString(data.Terminals[i].Avg_prep_secs), "1", 0, "C", false, 0, "")
				pdf.CellFormat(20, lineh, helper.IntToString(data.Terminals[i].Total_Orders), "1", 0, "C", false, 0, "")
				pdf.CellFormat(20, lineh, helper.IntToString(data.Terminals[i].Total_items), "1", 0, "C", false, 0, "")

				var calc1 float64 = float64(float64(data.Terminals[i].Stage1)/float64(data.Terminals[i].Total_Orders)) * 100
				var calc2 float64 = float64(float64(data.Terminals[i].Stage2)/float64(data.Terminals[i].Total_Orders)) * 100
				var calc3 float64 = float64(float64(data.Terminals[i].Stage3)/float64(data.Terminals[i].Total_Orders)) * 100
				var calc4 float64 = float64(float64(data.Terminals[i].Stage4)/float64(data.Terminals[i].Total_Orders)) * 100
				var calc5 float64 = float64(float64(data.Terminals[i].Stage5)/float64(data.Terminals[i].Total_Orders)) * 100
				var calc6 float64 = float64(float64(data.Terminals[i].Stage6)/float64(data.Terminals[i].Total_Orders)) * 100
				var calc7 float64 = float64(float64(data.Terminals[i].Stage7)/float64(data.Terminals[i].Total_Orders)) * 100
				pdf.CellFormat(25, lineh, helper.IntToString(data.Terminals[i].Stage1)+"("+helper.IntToString(int(calc1))+"%)", "1", 0, "C", false, 0, "")
				pdf.CellFormat(25, lineh, helper.IntToString(data.Terminals[i].Stage2)+"("+helper.IntToString(int(calc2))+"%)", "1", 0, "C", false, 0, "")
				pdf.CellFormat(25, lineh, helper.IntToString(data.Terminals[i].Stage3)+"("+helper.IntToString(int(calc3))+"%)", "1", 0, "C", false, 0, "")
				pdf.CellFormat(25, lineh, helper.IntToString(data.Terminals[i].Stage4)+"("+helper.IntToString(int(calc4))+"%)", "1", 0, "C", false, 0, "")
				pdf.CellFormat(25, lineh, helper.IntToString(data.Terminals[i].Stage5)+"("+helper.IntToString(int(calc5))+"%)", "1", 0, "C", false, 0, "")
				pdf.CellFormat(25, lineh, helper.IntToString(data.Terminals[i].Stage6)+"("+helper.IntToString(int(calc6))+"%)", "1", 0, "C", false, 0, "")
				pdf.CellFormat(25, lineh, helper.IntToString(data.Terminals[i].Stage7)+"("+helper.IntToString(int(calc7))+"%)", "1", 0, "C", false, 0, "")
				pdf.Ln(-1)
				for j := range data.Terminals[i].Profiles {
					pdf.SetFont("Arial", "", 8)
					pdf.CellFormat(35, lineh, "->"+data.Terminals[i].Profiles[j].Profile, "0", 0, "", false, 0, "")
					pdf.CellFormat(30, lineh, helper.IntToString(data.Terminals[i].Profiles[j].Avg_prep_mins)+":"+helper.IntToDoubleString(data.Terminals[i].Profiles[j].Avg_prep_secs), "1", 0, "C", false, 0, "")
					pdf.CellFormat(20, lineh, helper.IntToString(data.Terminals[i].Profiles[j].Total_Orders), "1", 0, "C", false, 0, "")
					pdf.CellFormat(20, lineh, helper.IntToString(data.Terminals[i].Profiles[j].Total_items), "1", 0, "C", false, 0, "")

					var calc1 float64 = float64(float64(data.Terminals[i].Profiles[j].Stage1)/float64(data.Terminals[i].Profiles[j].Total_Orders)) * 100
					var calc2 float64 = float64(float64(data.Terminals[i].Profiles[j].Stage2)/float64(data.Terminals[i].Profiles[j].Total_Orders)) * 100
					var calc3 float64 = float64(float64(data.Terminals[i].Profiles[j].Stage3)/float64(data.Terminals[i].Profiles[j].Total_Orders)) * 100
					var calc4 float64 = float64(float64(data.Terminals[i].Profiles[j].Stage4)/float64(data.Terminals[i].Profiles[j].Total_Orders)) * 100
					var calc5 float64 = float64(float64(data.Terminals[i].Profiles[j].Stage5)/float64(data.Terminals[i].Profiles[j].Total_Orders)) * 100
					var calc6 float64 = float64(float64(data.Terminals[i].Profiles[j].Stage6)/float64(data.Terminals[i].Profiles[j].Total_Orders)) * 100
					var calc7 float64 = float64(float64(data.Terminals[i].Profiles[j].Stage7)/float64(data.Terminals[i].Profiles[j].Total_Orders)) * 100
					pdf.CellFormat(25, lineh, helper.IntToString(data.Terminals[i].Profiles[j].Stage1)+"("+helper.IntToString(int(calc1))+"%)", "1", 0, "C", false, 0, "")
					pdf.CellFormat(25, lineh, helper.IntToString(data.Terminals[i].Profiles[j].Stage2)+"("+helper.IntToString(int(calc2))+"%)", "1", 0, "C", false, 0, "")
					pdf.CellFormat(25, lineh, helper.IntToString(data.Terminals[i].Profiles[j].Stage3)+"("+helper.IntToString(int(calc3))+"%)", "1", 0, "C", false, 0, "")
					pdf.CellFormat(25, lineh, helper.IntToString(data.Terminals[i].Profiles[j].Stage4)+"("+helper.IntToString(int(calc4))+"%)", "1", 0, "C", false, 0, "")
					pdf.CellFormat(25, lineh, helper.IntToString(data.Terminals[i].Profiles[j].Stage5)+"("+helper.IntToString(int(calc5))+"%)", "1", 0, "C", false, 0, "")
					pdf.CellFormat(25, lineh, helper.IntToString(data.Terminals[i].Profiles[j].Stage6)+"("+helper.IntToString(int(calc6))+"%)", "1", 0, "C", false, 0, "")
					pdf.CellFormat(25, lineh, helper.IntToString(data.Terminals[i].Profiles[j].Stage7)+"("+helper.IntToString(int(calc7))+"%)", "1", 0, "C", false, 0, "")
					pdf.Ln(-1)
				}
				pdf.Ln(10)
			}
		}

		_ = pdf.OutputFileAndClose("/tmp/report.pdf")
		ran := helper.RandStringRunes(10)
		awshelper.KDSUploadReport("/tmp/report.pdf", "kdsdailybackup", "kdsreports/"+ran+".pdf")
		return "kdsreports/" + ran + ".pdf"
		//_ = pdf.OutputFileAndClose("/tmp/report.pdf")
	}
	return "DONE"
}

func DataAccountSummaryWaiterReport(userid int, fromdate string, todate string) kdsmodels.KDSAccountSummaryMain {
	db := database.KDSConnection()
	main := kdsmodels.KDSAccountSummaryMain{}
	useraccounts := getUserAccountsFromUserID(userid, *db)
	for i := range useraccounts {
		terminals := getTerminalsFromAccountID(useraccounts[i].AccountID, *db)
		if len(terminals) > 0 {
			for j := range terminals {
				spew.Dump("CHECKING TERMINAL : " + terminals[j].Name)
				terminalentry := kdsmodels.KDSAccountSummaryMainTerminal{}
				account := getAccountFromAccountID(useraccounts[i].AccountID, *db)
				terminalentry.Terminal = terminals[j]
				terminalentry.Account = account
				result1 := res{}
				db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("AVG(prep_mins) DIV 1 as avg_prep_mins , ROUND(MOD(AVG(prep_mins) , 1) * 60) as avg_prep_secs , SUM(qty) as total_items").Where("terminal_id = ?", terminals[j].ID).Where("created >= ?", fromdate).Where("created <= ?", todate).First(&result1)
				if result1.Total_items > 0 {
					terminalentry.Avg_prep_mins = result1.Avg_prep_mins
					terminalentry.Avg_prep_secs = result1.Avg_prep_secs
					terminalentry.Total_items = result1.Total_items
					ressingle := resSingle{}
					db.Model(&kdsmodels.KDSExportOrders{}).Select("count(1) as single").Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created_at_on_kds >= ?", fromdate).Where("created_at_on_kds <= ?", todate).First(&ressingle)
					terminalentry.Total_Orders = ressingle.Single
					db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("COUNT(DISTINCT(kds_order_id)) as single").Where("prep_mins BETWEEN ? AND ?", 0, 2).Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created >= ?", fromdate).Where("created <= ?", todate).First(&ressingle)
					terminalentry.Stage1 = ressingle.Single
					db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("COUNT(DISTINCT(kds_order_id)) as single").Where("prep_mins BETWEEN ? AND ?", 0, 4).Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created >= ?", fromdate).Where("created <= ?", todate).First(&ressingle)
					terminalentry.Stage2 = ressingle.Single
					db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("COUNT(DISTINCT(kds_order_id)) as single").Where("prep_mins BETWEEN ? AND ?", 0, 6).Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created >= ?", fromdate).Where("created <= ?", todate).First(&ressingle)
					terminalentry.Stage3 = ressingle.Single
					db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("COUNT(DISTINCT(kds_order_id)) as single").Where("prep_mins BETWEEN ? AND ?", 0, 8).Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created >= ?", fromdate).Where("created <= ?", todate).First(&ressingle)
					terminalentry.Stage4 = ressingle.Single
					db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("COUNT(DISTINCT(kds_order_id)) as single").Where("prep_mins BETWEEN ? AND ?", 0, 10).Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created >= ?", fromdate).Where("created <= ?", todate).First(&ressingle)
					terminalentry.Stage5 = ressingle.Single
					db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("COUNT(DISTINCT(kds_order_id)) as single").Where("prep_mins BETWEEN ? AND ?", 0, 15).Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created >= ?", fromdate).Where("created <= ?", todate).First(&ressingle)
					terminalentry.Stage6 = ressingle.Single
					db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("COUNT(DISTINCT(kds_order_id)) as single").Where("prep_mins > ?", 15).Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created >= ?", fromdate).Where("created <= ?", todate).First(&ressingle)
					terminalentry.Stage7 = ressingle.Single

					// get all the waiter names
					// do above per waiter name
					result2 := []resSingleString{}
					result3 := []string{}
					result22 := []resSingleString{}
					db.Model(&kdsmodels.KDSExportOrders{}).Select("DISTINCT(waiter) as single").Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created_at_on_kds >= ?", fromdate).Where("created_at_on_kds <= ?", todate).Find(&result2)
					for t := range result2 {
						parts := strings.Split(result2[t].Single, "(")
						abc := strings.TrimSuffix(parts[0], "\n")
						abc = strings.TrimSpace(abc)
						result3 = append(result3, abc)
					}
					result3 = removeDuplicateStr(result3)
					for t := range result3 {
						abc := resSingleString{}
						abc.Single = result3[t]
						result22 = append(result22, abc)
					}
					result2 = result22

					for k := range result2 {
						profileentry := kdsmodels.KDSAccountSummaryMainTerminalProfile{}
						profileentry.Profile = result2[k].Single
						profileSubQuery1 := db.Model(&kdsmodels.KDSExportOrders{}).Select("id").Where("terminal_id = ?", terminalentry.Terminal.ID).Where("waiter like ?", profileentry.Profile+"%").Where("created >= ?", fromdate).Where("created <= ?", todate)
						db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("AVG(prep_mins) DIV 1 as avg_prep_mins , ROUND(MOD(AVG(prep_mins) , 1) * 60) as avg_prep_secs , SUM(qty) as total_items").Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created >= ?", fromdate).Where("created <= ?", todate).Where("kds_order_id IN (?)", profileSubQuery1).First(&result1)
						profileentry.Avg_prep_mins = result1.Avg_prep_mins
						profileentry.Avg_prep_secs = result1.Avg_prep_secs
						profileentry.Total_items = result1.Total_items
						db.Model(&kdsmodels.KDSExportOrders{}).Select("count(1) as single").Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created_at_on_kds >= ?", fromdate).Where("created_at_on_kds <= ?", todate).Where("waiter like ?", profileentry.Profile+"%").First(&ressingle)
						profileentry.Total_Orders = ressingle.Single

						db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("COUNT(DISTINCT(kds_order_id)) as single").Where("prep_mins BETWEEN ? AND ?", 0, 2).Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created >= ?", fromdate).Where("created <= ?", todate).Where("kds_order_id IN (?)", profileSubQuery1).First(&ressingle)
						profileentry.Stage1 = ressingle.Single
						db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("COUNT(DISTINCT(kds_order_id)) as single").Where("prep_mins BETWEEN ? AND ?", 0, 4).Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created >= ?", fromdate).Where("created <= ?", todate).Where("kds_order_id IN (?)", profileSubQuery1).First(&ressingle)
						profileentry.Stage2 = ressingle.Single
						db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("COUNT(DISTINCT(kds_order_id)) as single").Where("prep_mins BETWEEN ? AND ?", 0, 6).Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created >= ?", fromdate).Where("created <= ?", todate).Where("kds_order_id IN (?)", profileSubQuery1).First(&ressingle)
						profileentry.Stage3 = ressingle.Single
						db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("COUNT(DISTINCT(kds_order_id)) as single").Where("prep_mins BETWEEN ? AND ?", 0, 8).Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created >= ?", fromdate).Where("created <= ?", todate).Where("kds_order_id IN (?)", profileSubQuery1).First(&ressingle)
						profileentry.Stage4 = ressingle.Single
						db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("COUNT(DISTINCT(kds_order_id)) as single").Where("prep_mins BETWEEN ? AND ?", 0, 10).Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created >= ?", fromdate).Where("created <= ?", todate).Where("kds_order_id IN (?)", profileSubQuery1).First(&ressingle)
						profileentry.Stage5 = ressingle.Single
						db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("COUNT(DISTINCT(kds_order_id)) as single").Where("prep_mins BETWEEN ? AND ?", 0, 15).Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created >= ?", fromdate).Where("created <= ?", todate).Where("kds_order_id IN (?)", profileSubQuery1).First(&ressingle)
						profileentry.Stage6 = ressingle.Single
						db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("COUNT(DISTINCT(kds_order_id)) as single").Where("prep_mins > ?", 15).Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created >= ?", fromdate).Where("created <= ?", todate).Where("kds_order_id IN (?)", profileSubQuery1).First(&ressingle)
						profileentry.Stage7 = ressingle.Single

						terminalentry.Profiles = append(terminalentry.Profiles, profileentry)
					}
				}
				main.Terminals = append(main.Terminals, terminalentry)
			}
		}
	}

	return main
}

func removeDuplicateStr(strSlice []string) []string {
	allKeys := make(map[string]bool)
	list := []string{}
	for _, item := range strSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

func DataAccountSummaryReport(userid int, fromdate string, todate string) kdsmodels.KDSAccountSummaryMain {
	db := database.KDSConnection()
	main := kdsmodels.KDSAccountSummaryMain{}
	useraccounts := getUserAccountsFromUserID(userid, *db)
	for i := range useraccounts {
		terminals := getTerminalsFromAccountID(useraccounts[i].AccountID, *db)
		if len(terminals) > 0 {
			for j := range terminals {
				spew.Dump("CHECKING TERMINAL : " + terminals[j].Name)
				terminalentry := kdsmodels.KDSAccountSummaryMainTerminal{}
				account := getAccountFromAccountID(useraccounts[i].AccountID, *db)
				terminalentry.Terminal = terminals[j]
				terminalentry.Account = account
				result1 := res{}
				db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("AVG(prep_mins) DIV 1 as avg_prep_mins , ROUND(MOD(AVG(prep_mins) , 1) * 60) as avg_prep_secs , SUM(qty) as total_items").Where("terminal_id = ?", terminals[j].ID).Where("created >= ?", fromdate).Where("created <= ?", todate).First(&result1)
				if result1.Total_items > 0 {
					terminalentry.Avg_prep_mins = result1.Avg_prep_mins
					terminalentry.Avg_prep_secs = result1.Avg_prep_secs
					terminalentry.Total_items = result1.Total_items
					ressingle := resSingle{}
					db.Model(&kdsmodels.KDSExportOrders{}).Select("count(1) as single").Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created_at_on_kds >= ?", fromdate).Where("created_at_on_kds <= ?", todate).First(&ressingle)
					terminalentry.Total_Orders = ressingle.Single
					db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("COUNT(DISTINCT(kds_order_id)) as single").Where("prep_mins BETWEEN ? AND ?", 0, 2).Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created >= ?", fromdate).Where("created <= ?", todate).First(&ressingle)
					terminalentry.Stage1 = ressingle.Single
					db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("COUNT(DISTINCT(kds_order_id)) as single").Where("prep_mins BETWEEN ? AND ?", 0, 4).Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created >= ?", fromdate).Where("created <= ?", todate).First(&ressingle)
					terminalentry.Stage2 = ressingle.Single
					db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("COUNT(DISTINCT(kds_order_id)) as single").Where("prep_mins BETWEEN ? AND ?", 0, 6).Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created >= ?", fromdate).Where("created <= ?", todate).First(&ressingle)
					terminalentry.Stage3 = ressingle.Single
					db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("COUNT(DISTINCT(kds_order_id)) as single").Where("prep_mins BETWEEN ? AND ?", 0, 8).Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created >= ?", fromdate).Where("created <= ?", todate).First(&ressingle)
					terminalentry.Stage4 = ressingle.Single
					db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("COUNT(DISTINCT(kds_order_id)) as single").Where("prep_mins BETWEEN ? AND ?", 0, 10).Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created >= ?", fromdate).Where("created <= ?", todate).First(&ressingle)
					terminalentry.Stage5 = ressingle.Single
					db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("COUNT(DISTINCT(kds_order_id)) as single").Where("prep_mins BETWEEN ? AND ?", 0, 15).Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created >= ?", fromdate).Where("created <= ?", todate).First(&ressingle)
					terminalentry.Stage6 = ressingle.Single
					db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("COUNT(DISTINCT(kds_order_id)) as single").Where("prep_mins > ?", 15).Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created >= ?", fromdate).Where("created <= ?", todate).First(&ressingle)
					terminalentry.Stage7 = ressingle.Single

					// get all the profiles
					// do above per profile
					result2 := []resSingleString{}
					db.Model(&kdsmodels.KDSExportOrders{}).Select("DISTINCT(profile) as single").Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created_at_on_kds >= ?", fromdate).Where("created_at_on_kds <= ?", todate).Find(&result2)
					for k := range result2 {
						profileentry := kdsmodels.KDSAccountSummaryMainTerminalProfile{}
						profileentry.Profile = result2[k].Single
						profileSubQuery1 := db.Model(&kdsmodels.KDSExportOrders{}).Select("id").Where("terminal_id = ?", terminalentry.Terminal.ID).Where("profile = ?", profileentry.Profile).Where("created >= ?", fromdate).Where("created <= ?", todate)
						db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("AVG(prep_mins) DIV 1 as avg_prep_mins , ROUND(MOD(AVG(prep_mins) , 1) * 60) as avg_prep_secs , SUM(qty) as total_items").Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created >= ?", fromdate).Where("created <= ?", todate).Where("kds_order_id IN (?)", profileSubQuery1).First(&result1)
						profileentry.Avg_prep_mins = result1.Avg_prep_mins
						profileentry.Avg_prep_secs = result1.Avg_prep_secs
						profileentry.Total_items = result1.Total_items
						db.Model(&kdsmodels.KDSExportOrders{}).Select("count(1) as single").Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created_at_on_kds >= ?", fromdate).Where("created_at_on_kds <= ?", todate).Where("profile = ?", profileentry.Profile).First(&ressingle)
						profileentry.Total_Orders = ressingle.Single

						db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("COUNT(DISTINCT(kds_order_id)) as single").Where("prep_mins BETWEEN ? AND ?", 0, 2).Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created >= ?", fromdate).Where("created <= ?", todate).Where("kds_order_id IN (?)", profileSubQuery1).First(&ressingle)
						profileentry.Stage1 = ressingle.Single
						db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("COUNT(DISTINCT(kds_order_id)) as single").Where("prep_mins BETWEEN ? AND ?", 0, 4).Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created >= ?", fromdate).Where("created <= ?", todate).Where("kds_order_id IN (?)", profileSubQuery1).First(&ressingle)
						profileentry.Stage2 = ressingle.Single
						db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("COUNT(DISTINCT(kds_order_id)) as single").Where("prep_mins BETWEEN ? AND ?", 0, 6).Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created >= ?", fromdate).Where("created <= ?", todate).Where("kds_order_id IN (?)", profileSubQuery1).First(&ressingle)
						profileentry.Stage3 = ressingle.Single
						db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("COUNT(DISTINCT(kds_order_id)) as single").Where("prep_mins BETWEEN ? AND ?", 0, 8).Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created >= ?", fromdate).Where("created <= ?", todate).Where("kds_order_id IN (?)", profileSubQuery1).First(&ressingle)
						profileentry.Stage4 = ressingle.Single
						db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("COUNT(DISTINCT(kds_order_id)) as single").Where("prep_mins BETWEEN ? AND ?", 0, 10).Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created >= ?", fromdate).Where("created <= ?", todate).Where("kds_order_id IN (?)", profileSubQuery1).First(&ressingle)
						profileentry.Stage5 = ressingle.Single
						db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("COUNT(DISTINCT(kds_order_id)) as single").Where("prep_mins BETWEEN ? AND ?", 0, 15).Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created >= ?", fromdate).Where("created <= ?", todate).Where("kds_order_id IN (?)", profileSubQuery1).First(&ressingle)
						profileentry.Stage6 = ressingle.Single
						db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("COUNT(DISTINCT(kds_order_id)) as single").Where("prep_mins > ?", 15).Where("terminal_id = ?", terminalentry.Terminal.ID).Where("created >= ?", fromdate).Where("created <= ?", todate).Where("kds_order_id IN (?)", profileSubQuery1).First(&ressingle)
						profileentry.Stage7 = ressingle.Single

						terminalentry.Profiles = append(terminalentry.Profiles, profileentry)
					}
				}
				main.Terminals = append(main.Terminals, terminalentry)
			}
		}
	}

	return main
}

func GenerateAccountSummaryData(userid int, accountid int, fromdate string, todate string) kdsmodels.KDSAccountSummary {
	result := kdsmodels.KDSAccountSummary{}
	db := database.KDSConnection()
	result.Account = getAccountFromAccountID(accountid, *db)
	if result.Account.ID > 0 {
		result.Terminals = getTerminalsFromAccountID(result.Account.ID, *db)
		if len(result.Terminals) > 0 {
			for j := range result.Terminals {
				result1 := res{}
				sum := kdsmodels.KDSTerminalSummary{}
				db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("AVG(prep_mins) DIV 1 as avg_prep_mins , ROUND(MOD(AVG(prep_mins) , 1) * 60) as avg_prep_secs , SUM(qty) as total_items").Where("terminal_id = ?", result.Terminals[j].ID).Where("created >= ?", fromdate).Where("created <= ?", todate).First(&result1)
				sum.Avg_prep_mins = result1.Avg_prep_mins
				sum.Avg_prep_secs = result1.Avg_prep_secs
				sum.Terminal = result.Terminals[j]
				sum.Total_items = result1.Total_items
				config := getTerminalConfigFromTerminalID(sum.Terminal.ID, *db)
				late1 := 2
				late2 := 4
				if config.ID > 0 {
					kdsconfig := kdsmodels.KDSConfig{}
					wrong := json.Unmarshal([]byte(config.Config), &kdsconfig)
					if wrong == nil {
						if helper.StringToInt(kdsconfig.Colors.Late1time) > 0 {
							late1 = helper.StringToInt(kdsconfig.Colors.Late1time)
						}
						if helper.StringToInt(kdsconfig.Colors.Late2time) > 0 {
							late2 = helper.StringToInt(kdsconfig.Colors.Late2time)
						}
					}
				}
				ressingle := resSingle{}
				db.Model(&kdsmodels.KDSExportOrders{}).Select("count(1) as single").Where("terminal_id = ?", sum.Terminal.ID).Where("DATE(created_at_on_kds) >= ?", fromdate).Where("DATE(created_at_on_kds) <= ?", todate).First(&ressingle)
				sum.Total_Orders = ressingle.Single
				subQuery1 := db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("DISTINCT(kds_order_id)").Where("prep_mins > ?", late1).Where("terminal_id = ?", sum.Terminal.ID).Where("DATE(created) >= ?", fromdate).Where("DATE(created) <= ?", todate)
				db.Model(&kdsmodels.KDSExportOrders{}).Select("count(1) as single").Where("ID IN (?)", subQuery1).First(&ressingle)
				sum.Late1_Orders = ressingle.Single
				subQuery2 := db.Model(&kdsmodels.KDSExportItemsMySQL{}).Select("DISTINCT(kds_order_id)").Where("prep_mins > ?", late2).Where("terminal_id = ?", sum.Terminal.ID).Where("DATE(created) >= ?", fromdate).Where("DATE(created) <= ?", todate)
				db.Model(&kdsmodels.KDSExportOrders{}).Select("count(1) as single").Where("ID IN (?)", subQuery2).First(&ressingle)
				sum.Late2_Orders = ressingle.Single
				result.Summary = append(result.Summary, sum)
			}
		}
	}
	return result
}
func GenWaiterSummaryReport(userid int, fromdate string, todate string) string {
	data := DataAccountSummaryWaiterReport(userid, fromdate, todate)
	return GenericSummaryReport(data, fromdate, todate, "Waiter")
}
func GenericSummaryReport(data kdsmodels.KDSAccountSummaryMain, fromdate string, todate string, summarytype string) string {
	var lineh float64 = 10
	if len(data.Terminals) > 0 {
		pdf := PDFGENHeader("Summary Report", fromdate, todate)
		lastid := 0
		for i := range data.Terminals {
			if lastid != data.Terminals[i].Account.ID {
				pdf.AddPage()
				pdf.SetFont("Arial", "B", 10)
				pdf.Cell(0, 30, "LOCATION : "+data.Terminals[i].Account.Name+" ("+summarytype+" Summary Report"+" For "+fromdate+" - "+todate+")")
				pdf.Ln(30)
				pdf.CellFormat(35, lineh, "KDS", "1", 0, "", false, 0, "")
				pdf.CellFormat(30, lineh, "AVG Prep", "1", 0, "C", false, 0, "")
				pdf.CellFormat(20, lineh, "Orders", "1", 0, "C", false, 0, "")
				pdf.CellFormat(20, lineh, "Items", "1", 0, "C", false, 0, "")

				pdf.CellFormat(25, lineh, "0-2M", "1", 0, "C", false, 0, "")
				pdf.CellFormat(25, lineh, "0-4M", "1", 0, "C", false, 0, "")
				pdf.CellFormat(25, lineh, "0-6M", "1", 0, "C", false, 0, "")
				pdf.CellFormat(25, lineh, "0-8M", "1", 0, "C", false, 0, "")
				pdf.CellFormat(25, lineh, "0-10M", "1", 0, "C", false, 0, "")
				pdf.CellFormat(25, lineh, "0-15M", "1", 0, "C", false, 0, "")
				pdf.CellFormat(25, lineh, "15M+", "1", 0, "C", false, 0, "")
				pdf.Ln(-1)
			}
			lastid = data.Terminals[i].Account.ID
			if data.Terminals[i].Total_Orders > 0 {
				pdf.SetFont("Arial", "B", 8)
				pdf.CellFormat(35, lineh, data.Terminals[i].Terminal.Name, "0", 0, "", false, 0, "")
				pdf.SetFont("Arial", "", 8)
				pdf.CellFormat(30, lineh, helper.IntToString(data.Terminals[i].Avg_prep_mins)+":"+helper.IntToDoubleString(data.Terminals[i].Avg_prep_secs), "1", 0, "C", false, 0, "")
				pdf.CellFormat(20, lineh, helper.IntToString(data.Terminals[i].Total_Orders), "1", 0, "C", false, 0, "")
				pdf.CellFormat(20, lineh, helper.IntToString(data.Terminals[i].Total_items), "1", 0, "C", false, 0, "")

				var calc1 float64 = float64(float64(data.Terminals[i].Stage1)/float64(data.Terminals[i].Total_Orders)) * 100
				var calc2 float64 = float64(float64(data.Terminals[i].Stage2)/float64(data.Terminals[i].Total_Orders)) * 100
				var calc3 float64 = float64(float64(data.Terminals[i].Stage3)/float64(data.Terminals[i].Total_Orders)) * 100
				var calc4 float64 = float64(float64(data.Terminals[i].Stage4)/float64(data.Terminals[i].Total_Orders)) * 100
				var calc5 float64 = float64(float64(data.Terminals[i].Stage5)/float64(data.Terminals[i].Total_Orders)) * 100
				var calc6 float64 = float64(float64(data.Terminals[i].Stage6)/float64(data.Terminals[i].Total_Orders)) * 100
				var calc7 float64 = float64(float64(data.Terminals[i].Stage7)/float64(data.Terminals[i].Total_Orders)) * 100
				pdf.CellFormat(25, lineh, helper.IntToString(data.Terminals[i].Stage1)+"("+helper.IntToString(int(calc1))+"%)", "1", 0, "C", false, 0, "")
				pdf.CellFormat(25, lineh, helper.IntToString(data.Terminals[i].Stage2)+"("+helper.IntToString(int(calc2))+"%)", "1", 0, "C", false, 0, "")
				pdf.CellFormat(25, lineh, helper.IntToString(data.Terminals[i].Stage3)+"("+helper.IntToString(int(calc3))+"%)", "1", 0, "C", false, 0, "")
				pdf.CellFormat(25, lineh, helper.IntToString(data.Terminals[i].Stage4)+"("+helper.IntToString(int(calc4))+"%)", "1", 0, "C", false, 0, "")
				pdf.CellFormat(25, lineh, helper.IntToString(data.Terminals[i].Stage5)+"("+helper.IntToString(int(calc5))+"%)", "1", 0, "C", false, 0, "")
				pdf.CellFormat(25, lineh, helper.IntToString(data.Terminals[i].Stage6)+"("+helper.IntToString(int(calc6))+"%)", "1", 0, "C", false, 0, "")
				pdf.CellFormat(25, lineh, helper.IntToString(data.Terminals[i].Stage7)+"("+helper.IntToString(int(calc7))+"%)", "1", 0, "C", false, 0, "")
				pdf.Ln(-1)
				for j := range data.Terminals[i].Profiles {
					pdf.SetFont("Arial", "", 8)
					pdf.CellFormat(35, lineh, "->"+data.Terminals[i].Profiles[j].Profile, "0", 0, "", false, 0, "")
					pdf.CellFormat(30, lineh, helper.IntToString(data.Terminals[i].Profiles[j].Avg_prep_mins)+":"+helper.IntToDoubleString(data.Terminals[i].Profiles[j].Avg_prep_secs), "1", 0, "C", false, 0, "")
					pdf.CellFormat(20, lineh, helper.IntToString(data.Terminals[i].Profiles[j].Total_Orders), "1", 0, "C", false, 0, "")
					pdf.CellFormat(20, lineh, helper.IntToString(data.Terminals[i].Profiles[j].Total_items), "1", 0, "C", false, 0, "")

					var calc1 float64 = float64(float64(data.Terminals[i].Profiles[j].Stage1)/float64(data.Terminals[i].Profiles[j].Total_Orders)) * 100
					var calc2 float64 = float64(float64(data.Terminals[i].Profiles[j].Stage2)/float64(data.Terminals[i].Profiles[j].Total_Orders)) * 100
					var calc3 float64 = float64(float64(data.Terminals[i].Profiles[j].Stage3)/float64(data.Terminals[i].Profiles[j].Total_Orders)) * 100
					var calc4 float64 = float64(float64(data.Terminals[i].Profiles[j].Stage4)/float64(data.Terminals[i].Profiles[j].Total_Orders)) * 100
					var calc5 float64 = float64(float64(data.Terminals[i].Profiles[j].Stage5)/float64(data.Terminals[i].Profiles[j].Total_Orders)) * 100
					var calc6 float64 = float64(float64(data.Terminals[i].Profiles[j].Stage6)/float64(data.Terminals[i].Profiles[j].Total_Orders)) * 100
					var calc7 float64 = float64(float64(data.Terminals[i].Profiles[j].Stage7)/float64(data.Terminals[i].Profiles[j].Total_Orders)) * 100
					pdf.CellFormat(25, lineh, helper.IntToString(data.Terminals[i].Profiles[j].Stage1)+"("+helper.IntToString(int(calc1))+"%)", "1", 0, "C", false, 0, "")
					pdf.CellFormat(25, lineh, helper.IntToString(data.Terminals[i].Profiles[j].Stage2)+"("+helper.IntToString(int(calc2))+"%)", "1", 0, "C", false, 0, "")
					pdf.CellFormat(25, lineh, helper.IntToString(data.Terminals[i].Profiles[j].Stage3)+"("+helper.IntToString(int(calc3))+"%)", "1", 0, "C", false, 0, "")
					pdf.CellFormat(25, lineh, helper.IntToString(data.Terminals[i].Profiles[j].Stage4)+"("+helper.IntToString(int(calc4))+"%)", "1", 0, "C", false, 0, "")
					pdf.CellFormat(25, lineh, helper.IntToString(data.Terminals[i].Profiles[j].Stage5)+"("+helper.IntToString(int(calc5))+"%)", "1", 0, "C", false, 0, "")
					pdf.CellFormat(25, lineh, helper.IntToString(data.Terminals[i].Profiles[j].Stage6)+"("+helper.IntToString(int(calc6))+"%)", "1", 0, "C", false, 0, "")
					pdf.CellFormat(25, lineh, helper.IntToString(data.Terminals[i].Profiles[j].Stage7)+"("+helper.IntToString(int(calc7))+"%)", "1", 0, "C", false, 0, "")
					pdf.Ln(-1)
				}
				pdf.Ln(10)
			}
		}

		_ = pdf.OutputFileAndClose("/tmp/report.pdf")
		ran := helper.RandStringRunes(10)
		awshelper.KDSUploadReport("/tmp/report.pdf", "kdsdailybackup", "kdsreports/"+ran+".pdf")
		return "kdsreports/" + ran + ".pdf"
		//_ = pdf.OutputFileAndClose("/tmp/report.pdf")
	}
	return "DONE"
}
