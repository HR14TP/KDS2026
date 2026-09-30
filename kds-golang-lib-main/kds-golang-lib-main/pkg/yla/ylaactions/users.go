package ylaactions

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/reverent911/kds-golang-lib/pkg/generic/database"
	"github.com/reverent911/kds-golang-lib/pkg/generic/responses"
	"github.com/reverent911/kds-golang-lib/pkg/yla/ylamodels"
)

func YLALoginUser(input ylamodels.YLALoginUser) events.APIGatewayProxyResponse {
	db := database.YLAConnection()
	ylauser := YLAGetUserByEmailAndPassword(input.Email, input.Password, db)
	return responses.GenAPIResponse(200, ylauser)
}
