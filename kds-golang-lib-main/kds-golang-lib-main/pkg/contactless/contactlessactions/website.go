package contactlessactions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/reverent911/kds-golang-lib/pkg/contactless/contactlessmodels"
	"github.com/reverent911/kds-golang-lib/pkg/generic/helper"
)

// MenuImportCategory ... Menu Import Category
func MenuImportCategory(site string, name string, baseid int) contactlessmodels.ContactlessCategory {
	url := site + "/wp-json/kseries/v1/category" + "?catid=" + helper.IntToString(baseid) + "&catname=" + name
	req, err := http.NewRequest("GET", url, nil)
	HandleErrors(err)
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 10
	client := retryClient.StandardClient()
	resp, err := client.Do(req)
	HandleErrors(err)
	body, _ := ioutil.ReadAll(resp.Body)
	category := contactlessmodels.ContactlessCategory{}
	err = json.Unmarshal([]byte(body), &category)
	HandleErrors(err)
	ShowDebug("********")
	ShowDebug("Creating Category : " + name)
	ShowDebug(category)
	ShowDebug("========")
	return category
}

// GetContactlessIDSFromWooCommerce ... Get Contactless IDS From WooCommerce
func GetContactlessIDSFromWooCommerce(url string) contactlessmodels.ContactlessToken {
	siteurl := url + "/wp-json/kseries/v1/token"
	req, err := http.NewRequest("GET", siteurl, nil)
	HandleErrors(err)
	client := &http.Client{}
	resp, err := client.Do(req)
	HandleErrors(err)
	body, _ := ioutil.ReadAll(resp.Body)
	ShowDebug(string(body))
	token := contactlessmodels.ContactlessToken{}
	err = json.Unmarshal([]byte(body), &token)
	HandleErrors(err)
	return token
}

// MenuCreateAttribute ... Menu Create Attribute
func MenuCreateAttribute(site string, name string) contactlessmodels.ContactlessAttribute {
	url := site + "/wp-json/kseries/v1/attribute" + "?name=" + url.QueryEscape(name)
	req, err := http.NewRequest("GET", url, nil)
	HandleErrors(err)
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 10
	client := retryClient.StandardClient()
	resp, err := client.Do(req)
	HandleErrors(err)
	body, _ := ioutil.ReadAll(resp.Body)
	attr := contactlessmodels.ContactlessAttribute{}
	err = json.Unmarshal([]byte(body), &attr)
	HandleErrors(err)
	ShowDebug("********")
	ShowDebug("Creating Attribute : " + name)
	ShowDebug(attr)
	ShowDebug("========")
	return attr
}

// MenuCreateCustomAttribute ... Menu Create Custom Attribute
func MenuCreateCustomAttribute(site string, attrid string, thetype string, required bool) contactlessmodels.ContactlessAttributeTerm {
	var test = helper.StringToInt(attrid)
	term := contactlessmodels.ContactlessAttributeTerm{}
	if test > 0 {
		url := site + "/wp-json/kseries/v1/customterm" + "?attrid=" + attrid + "&type=" + thetype + "&required=" + strconv.FormatBool(required)
		req, err := http.NewRequest("GET", url, nil)
		HandleErrors(err)
		retryClient := retryablehttp.NewClient()
		retryClient.RetryMax = 10
		client := retryClient.StandardClient()
		resp, err := client.Do(req)
		HandleErrors(err)
		body, _ := ioutil.ReadAll(resp.Body)
		err = json.Unmarshal([]byte(body), &term)
		HandleErrors(err)
		ShowDebug("********")
		ShowDebug("Creating Custom Attribute")
		ShowDebug(string(body))
		ShowDebug("========")
	}
	return term
}

// MenuCreateSimpleItem ... Create simple item on woocommerce
func MenuCreateSimpleItem(site string, sku string, name string, price string, catid int) contactlessmodels.ContactlessSimpleItem {
	url := site + "/wp-json/kseries/v1/simpleitem" + "?catid=" + helper.IntToString(catid) + "&name=" + url.QueryEscape(name) + "&sku=" + sku + "&price=" + price
	req, err := http.NewRequest("GET", url, nil)
	HandleErrors(err)
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 10
	client := retryClient.StandardClient()
	resp, err := client.Do(req)
	HandleErrors(err)
	body, _ := ioutil.ReadAll(resp.Body)
	item := contactlessmodels.ContactlessSimpleItem{}
	err = json.Unmarshal([]byte(body), &item)
	HandleErrors(err)
	ShowDebug("********")
	ShowDebug("Creating Simple Item : " + name + ":" + sku)
	ShowDebug(item)
	ShowDebug("========")
	return item
}

// MenuCreateVariableItem ... Menu Create Variable Item
func MenuCreateVariableItem(site string, sku string, name string, price string, attrs []int, catid int) contactlessmodels.ContactlessSimpleItem {
	st := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(attrs)), ","), "[]")
	url := site + "/wp-json/kseries/v1/variableitem" + "?catid=" + helper.IntToString(catid) + "&attrs=" + url.QueryEscape(st) + "&name=" + url.QueryEscape(name) + "&sku=" + sku + "&price=" + price
	req, err := http.NewRequest("GET", url, nil)
	HandleErrors(err)
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 10
	client := retryClient.StandardClient()
	client.Timeout = 300 * time.Second
	resp, err := client.Do(req)
	HandleErrors(err)
	body, _ := ioutil.ReadAll(resp.Body)
	item := contactlessmodels.ContactlessSimpleItem{}
	err = json.Unmarshal([]byte(body), &item)
	HandleErrors(err)
	ShowDebug("********")
	ShowDebug("Creating Variable Item : " + name + ":" + sku)
	ShowDebug(item)
	ShowDebug("========")
	return item
}

// MenuCreateAttributeTerm ... Menu Create Attribute Term
func MenuCreateAttributeTerm(site string, name string, price string, attrid string, sku string) contactlessmodels.ContactlessAttributeTerm {
	url := site + "/wp-json/kseries/v1/term" + "?name=" + url.QueryEscape(name) + "&price=" + price + "&attrid=" + attrid + "&sku=" + sku
	req, err := http.NewRequest("GET", url, nil)
	HandleErrors(err)
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 10
	client := retryClient.StandardClient()
	resp, err := client.Do(req)
	HandleErrors(err)
	body, _ := ioutil.ReadAll(resp.Body)
	term := contactlessmodels.ContactlessAttributeTerm{}
	err = json.Unmarshal([]byte(body), &term)
	HandleErrors(err)
	ShowDebug("********")
	ShowDebug("Creating Attribute Term : " + name + ":" + sku)
	ShowDebug(term)
	ShowDebug("========")
	return term
}

// MenuAssignCustomMeta ... Menu Assign Variable Meta
func MenuAssignCustomMeta(site string, prodid int, attrs []int) {
	st := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(attrs)), ","), "[]")
	url := site + "/wp-json/kseries/v1/assignmeta" + "?prodid=" + helper.IntToString(prodid) + "&meta=" + url.QueryEscape(st)
	req, err := http.NewRequest("GET", url, nil)
	HandleErrors(err)
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 10
	client := retryClient.StandardClient()
	client.Timeout = 300 * time.Second
	resp, err := client.Do(req)
	HandleErrors(err)
	body, _ := ioutil.ReadAll(resp.Body)
	ShowDebug("********")
	ShowDebug("Assign Custom Meta")
	ShowDebug(string(body))
	ShowDebug("========")
}

func doItemChecks(url string) {
	for k, v := range Globict {
		s := strconv.Itoa(k)
		siteurl := url + "/wp-json/kseries/v1/updatecategoryitems?catid=" + s + "&skus=" + v
		req, err := http.NewRequest("GET", siteurl, nil)
		HandleErrors(err)
		client := &http.Client{}
		resp, err := client.Do(req)
		HandleErrors(err)
		body, _ := ioutil.ReadAll(resp.Body)
		ShowDebug(string(body))
	}
}

// MenuUpdateDescriptionUpdate ... Post update 2 wordpress for item description
func MenuUpdateDescriptionUpdate(site string, sku string, desc string) {
	pack := contactlessmodels.ContactLessDescriptionUpdate{}
	pack.Description = desc
	pack.SKU = sku
	b, _ := json.Marshal(pack)
	url := site + "/wp-json/kseries/v1/description"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
	HandleErrors(err)
	client := &http.Client{}
	resp, err := client.Do(req)
	HandleErrors(err)
	body, _ := ioutil.ReadAll(resp.Body)
	ShowDebug(string(body))
}

// MenuUpdateSimpleItem ... Update simple item on woocommerce
func MenuUpdateSimpleItem(site string, sku string, name string, price string) contactlessmodels.ContactlessSimpleItem {
	url := site + "/wp-json/kseries/v1/updatesimpleitem" + "?&name=" + url.QueryEscape(name) + "&sku=" + sku + "&price=" + price
	req, err := http.NewRequest("GET", url, nil)
	HandleErrors(err)
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 10
	client := retryClient.StandardClient()
	resp, err := client.Do(req)
	HandleErrors(err)
	body, _ := ioutil.ReadAll(resp.Body)
	item := contactlessmodels.ContactlessSimpleItem{}
	err = json.Unmarshal([]byte(body), &item)
	HandleErrors(err)
	return item
}

// UpdateContactlessJobStatus ... Update Contactless Job Status
func UpdateContactlessJobStatus(url string, jobtype string, jobstatus string) {
	siteurl := url + "/wp-json/kseries/v1/jobdone?status=" + jobstatus + "&type=" + jobtype
	req, err := http.NewRequest("GET", siteurl, nil)
	HandleErrors(err)
	client := &http.Client{}
	resp, err := client.Do(req)
	HandleErrors(err)
	body, _ := ioutil.ReadAll(resp.Body)
	ShowDebug(string(body))
}

// Start Maintenance Mode
func WebStartMaintenance(url string) {
	siteurl := url + "/wp-json/kseries/v1/startmaintenance"
	ShowDebug(siteurl)
	req, err := http.NewRequest("GET", siteurl, nil)
	HandleErrors(err)
	client := &http.Client{}
	resp, err := client.Do(req)
	HandleErrors(err)
	body, _ := ioutil.ReadAll(resp.Body)
	ShowDebug(string(body))
}

// End Maintenance Mode
func WebEndMaintenance(url string) {
	siteurl := url + "/wp-json/kseries/v1/stopmaintenance"
	ShowDebug(siteurl)
	req, err := http.NewRequest("GET", siteurl, nil)
	HandleErrors(err)
	client := &http.Client{}
	resp, err := client.Do(req)
	HandleErrors(err)
	body, _ := ioutil.ReadAll(resp.Body)
	ShowDebug(string(body))
}

// WebsiteCreateAttribute ... Menu Create Attribute
func WebsiteCreateAttribute(site string, name string) contactlessmodels.ContactlessAttribute {
	url := site + "/wp-json/kseries/v1/attribute" + "?name=" + url.QueryEscape(name)
	req, err := http.NewRequest("GET", url, nil)
	HandleErrors(err)
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 10
	client := retryClient.StandardClient()
	resp, err := client.Do(req)
	HandleErrors(err)
	body, _ := ioutil.ReadAll(resp.Body)
	attr := contactlessmodels.ContactlessAttribute{}
	err = json.Unmarshal([]byte(body), &attr)
	HandleErrors(err)
	return attr
}

// WebsiteCreateAttributeTerm ... Menu Create Attribute Term
func WebsiteCreateAttributeTerm(site string, name string, price string, attrid string, sku string) contactlessmodels.ContactlessAttributeTerm {
	url := site + "/wp-json/kseries/v1/term" + "?name=" + url.QueryEscape(name) + "&price=" + price + "&attrid=" + attrid + "&sku=" + sku
	ShowDebug("WebsiteCreateAttributeTerm")
	ShowDebug(url)
	req, err := http.NewRequest("GET", url, nil)
	HandleErrors(err)
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 10
	client := retryClient.StandardClient()
	resp, err := client.Do(req)
	HandleErrors(err)
	body, _ := ioutil.ReadAll(resp.Body)
	term := contactlessmodels.ContactlessAttributeTerm{}
	err = json.Unmarshal([]byte(body), &term)
	HandleErrors(err)
	return term
}

// WebsiteCreateCustomAttributeCombinedWithNameAndSKU ... Menu Create Custom Attribute
func WebsiteCreateCustomAttributeCombinedWithNameAndSKU(site string, attrid string, thetype string, required bool, name string, sku string) contactlessmodels.ContactlessAttributeTerm {
	var test = helper.StringToInt(attrid)
	term := contactlessmodels.ContactlessAttributeTerm{}
	if test > 0 {
		url := site + "/wp-json/kseries/v1/customtermcombined" + "?attrid=" + attrid + "&sku=" + sku + "&name=" + name + "&type=" + thetype + "&required=" + strconv.FormatBool(required)
		req, err := http.NewRequest("GET", url, nil)
		HandleErrors(err)
		retryClient := retryablehttp.NewClient()
		retryClient.RetryMax = 10
		client := retryClient.StandardClient()
		resp, err := client.Do(req)
		HandleErrors(err)
		body, _ := ioutil.ReadAll(resp.Body)

		err = json.Unmarshal([]byte(body), &term)
		HandleErrors(err)
		ShowDebug("********")
		ShowDebug("CreateCustomAttributeCombinedWithNameAndSKU : " + name + ":" + sku)
		fmt.Println(string(body))
		ShowDebug("========")
	}
	return term
}

// ProcessVariableItem ... Create Variable Item
func ProcessNewFullVariableItem(theurl string, sku string, name string, price string, attrs []int, baseid int) contactlessmodels.ContactlessSimpleItem {
	item := MenuCreateVariableItem(theurl, sku, name, price, attrs, baseid)
	Globict[baseid] += sku + ","
	return item
}

// WebsiteAssignCustomMetaSKU ... Menu Assign Variable Meta
func WebsiteAssignCustomMetaSKU(site string, sku string, attrs []int) {
	st := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(attrs)), ","), "[]")
	url := site + "/wp-json/kseries/v1/assignmetasku" + "?sku=" + sku + "&meta=" + url.QueryEscape(st)
	req, err := http.NewRequest("GET", url, nil)
	HandleErrors(err)
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 10
	client := retryClient.StandardClient()
	client.Timeout = 300 * time.Second
	resp, err := client.Do(req)
	HandleErrors(err)
	body, _ := ioutil.ReadAll(resp.Body)
	ShowDebug("********")
	ShowDebug("Assign Custom Meta SKU : " + sku)
	fmt.Println(string(body))
	ShowDebug("========")

}

// WebsiteUnlinkProducts ... Menu Assign Variable Meta
func WebsiteUnlinkProducts(site string) {
	url := site + "/wp-json/kseries/v1/unlink" + "?orderid=123"
	req, err := http.NewRequest("GET", url, nil)
	HandleErrors(err)
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 10
	client := retryClient.StandardClient()
	client.Timeout = 300 * time.Second
	resp, err := client.Do(req)
	HandleErrors(err)
	body, _ := ioutil.ReadAll(resp.Body)
	ShowDebug("********")
	ShowDebug("WebsiteUnlinkProducts : ")
	fmt.Println(string(body))
	ShowDebug("========")
}

type ProductSKU struct {
	SKU string `json:"sku"`
}

func GetAllProducts(site string) {
	url := site + "/wp-json/kseries/v1/getallproducts"
	req, err := http.NewRequest("GET", url, nil)
	HandleErrors(err)
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 10
	client := retryClient.StandardClient()
	client.Timeout = 300 * time.Second
	resp, err := client.Do(req)
	HandleErrors(err)
	body, _ := ioutil.ReadAll(resp.Body)
	var products []ProductSKU
	if err := json.Unmarshal([]byte(body), &products); err != nil {
		log.Fatal(err)
	}
	for i := range products {
		SingleItemMenuImportPriceOnly(site, products[i].SKU)
	}
}
