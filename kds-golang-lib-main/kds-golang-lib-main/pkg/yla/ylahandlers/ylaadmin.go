package ylahandlers

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/reverent911/kds-golang-lib/pkg/generic/helper"
	"github.com/reverent911/kds-golang-lib/pkg/generic/responses"
	"github.com/reverent911/kds-golang-lib/pkg/yla/ylaactions"
	"github.com/reverent911/kds-golang-lib/pkg/yla/ylamodels"
)

// YLAAdminLoginHandler ... YLAAdminLoginHandler
func YLAAdminLoginHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if request.HTTPMethod == "POST" {
		input := ylamodels.YLALoginUser{}
		err := json.Unmarshal([]byte(request.Body), &input)
		if err != nil {
			panic(err)
		}
		return ylaactions.YLALoginUser(input), nil
	}
	return responses.GenAPIResponse(200, "YLAAdminLoginHandler"), nil
}

// YLAAdminItemsHandler ... YLAAdminItemsHandler
func YLAAdminItemsHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if request.HTTPMethod == "POST" {
		input := ylamodels.YLALoginUser{}
		err := json.Unmarshal([]byte(request.Body), &input)
		if err != nil {
			panic(err)
		}
		return ylaactions.YLALoginUser(input), nil
	} else if request.HTTPMethod == "GET" {
		ikentooid := helper.StringToInt(request.QueryStringParameters["ikentooid"])
		itemid := helper.StringToInt(request.QueryStringParameters["itemid"])
		if itemid > 0 && ikentooid > 0 {
			//get full item with sequences

		} else if ikentooid > 0 {
			//get business items

		}
		fmt.Println(ikentooid)
		fmt.Println(itemid)
	}
	return responses.GenAPIResponse(200, "YLAAdminLoginHandler"), nil
}
