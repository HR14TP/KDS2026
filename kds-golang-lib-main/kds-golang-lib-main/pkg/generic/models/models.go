package models

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
)

type LambdaInput struct {
	Request events.APIGatewayProxyRequest
	Context context.Context
}

func (p *LambdaInput) GetPathCustom() string {
	action := p.Request.QueryStringParameters["action"]
	return action
}
