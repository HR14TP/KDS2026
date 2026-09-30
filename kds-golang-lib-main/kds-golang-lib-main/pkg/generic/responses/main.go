package responses

import (
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
)

type Proxy struct {
	Data interface{} `json:"data"`
	Code int         `json:"code"`
}

/*GenAPIResponse generates API response*/
func GenNonCORSAPIResponse(code int, data interface{}) events.APIGatewayProxyResponse {
	proxyHeaders := make(map[string]string)
	proxyHeaders["Content-Type"] = "application/json; charset=utf-8"
	proxyHeaders["Cache-Control"] = "no-cache"
	//proxyHeaders["Access-Control-Allow-Origin"] = "*"
	proxyresponse := &Proxy{Data: data, Code: code}
	proxybody, _ := json.Marshal(proxyresponse)
	var response = events.APIGatewayProxyResponse{
		Body:       string(proxybody[:]),
		StatusCode: code,
		Headers:    proxyHeaders,
	}
	return response
}

/*GenAPIResponse generates API response*/
func GenLambdaURLResponse(code int, data interface{}) events.APIGatewayProxyResponse {
	proxyHeaders := make(map[string]string)
	proxyHeaders["Content-Type"] = "application/json; charset=utf-8"
	proxyHeaders["Cache-Control"] = "no-cache"
	proxyHeaders["API-Version"] = "LambdaV1"
	proxyresponse := &Proxy{Data: data, Code: code}
	proxybody, _ := json.Marshal(proxyresponse)
	var response = events.APIGatewayProxyResponse{
		Body:       string(proxybody[:]),
		StatusCode: code,
		Headers:    proxyHeaders,
	}
	return response
}

/*GenAPIResponse generates API response*/
func GenAPIResponse(code int, data interface{}) events.APIGatewayProxyResponse {
	proxyHeaders := make(map[string]string)
	proxyHeaders["Content-Type"] = "application/json; charset=utf-8"
	proxyHeaders["Cache-Control"] = "no-cache"
	proxyHeaders["Access-Control-Allow-Origin"] = "*"
	proxyresponse := &Proxy{Data: data, Code: code}
	proxybody, _ := json.Marshal(proxyresponse)
	var response = events.APIGatewayProxyResponse{
		Body:       string(proxybody[:]),
		StatusCode: code,
		Headers:    proxyHeaders,
	}
	return response
}

// GenHTMLAPIResponse .. Generate HTML Response
func GenHTMLAPIResponse(code int, data string) events.APIGatewayProxyResponse {
	proxyHeaders := make(map[string]string)
	proxyHeaders["Content-Type"] = "text/html; charset=utf-8"
	proxyHeaders["Cache-Control"] = "no-cache"
	proxyHeaders["Access-Control-Allow-Origin"] = "*"
	var response = events.APIGatewayProxyResponse{
		Body:       string(data),
		StatusCode: code,
		Headers:    proxyHeaders,
	}
	return response
}

/*GenLambdaHTMLResponse generates HTML response*/
func GenLambdaHTMLResponse(code int, data string) events.APIGatewayProxyResponse {
	proxyHeaders := make(map[string]string)
	proxyHeaders["Content-Type"] = "text/html; charset=utf-8"
	proxyHeaders["Cache-Control"] = "no-cache"
	proxyHeaders["API-Version"] = "LambdaV1"
	var response = events.APIGatewayProxyResponse{
		Body:       string(data),
		StatusCode: code,
		Headers:    proxyHeaders,
	}
	return response
}
