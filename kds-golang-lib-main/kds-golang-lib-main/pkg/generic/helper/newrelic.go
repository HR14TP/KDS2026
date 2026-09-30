package helper

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func InitNewRelicDebug(appname string, appkey string) *newrelic.Application {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName(appname),
		newrelic.ConfigLicense(appkey),
		newrelic.ConfigDebugLogger(os.Stdout),
		newrelic.ConfigDistributedTracerEnabled(true),
		newrelic.ConfigEnabled(true),
	)
	if err != nil {
		fmt.Println(err.Error())
	}
	err = app.WaitForConnection(10 * time.Second)
	if err != nil {
		fmt.Println(err.Error())
	}
	return app
}

func InitNewRelic(appname string, appkey string) *newrelic.Application {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName(appname),
		newrelic.ConfigLicense(appkey),
		//newrelic.ConfigDebugLogger(os.Stdout),
		newrelic.ConfigDistributedTracerEnabled(true),
		newrelic.ConfigEnabled(true),
	)
	if err != nil {
		fmt.Println(err.Error())
	}
	err = app.WaitForConnection(10 * time.Second)
	if err != nil {
		fmt.Println(err.Error())
	}
	return app
}

func InitNewRelicNew(appname string, appkey string) (*newrelic.Application, error) {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName(appname),
		newrelic.ConfigLicense(appkey),
		//newrelic.ConfigDebugLogger(os.Stdout),
		newrelic.ConfigDistributedTracerEnabled(true),
		newrelic.ConfigEnabled(true),
	)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	err = app.WaitForConnection(10 * time.Second)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	return app, nil
}

func CaptureNewRelicEvent(app *newrelic.Application, eventName string, eventData map[string]interface{}) {
	app.RecordCustomEvent(eventName, eventData)
}

func CaptureServerlessHit(app *newrelic.Application, request events.APIGatewayProxyRequest) {
	app.RecordCustomEvent("APIGATEWAY", map[string]interface{}{
		"Path":   request.Path,
		"Method": request.HTTPMethod,
		"Body":   request.Body,
	})
}

func CreateTransactionNewRelic(app *newrelic.Application, transactionName string) *newrelic.Transaction {
	return app.StartTransaction(transactionName)
}

func CaptureNewRelicMetric(app *newrelic.Application, metricName string, metricValue float64) {
	app.RecordCustomEvent("SPEED", map[string]interface{}{
		"Type": metricName,
		"Time": int(metricValue),
	})
	app.RecordCustomMetric(metricName, metricValue)
	log.Printf("%v: %v\n", metricName, metricValue)
}

func CaptureNewRelicError(app *newrelic.Application, er error) {
	if er != nil {
		app.RecordCustomEvent("ERROR", map[string]interface{}{
			"Type":   "ERROR",
			"Detail": er.Error(),
		})
	}
}
func CaptureNewRelicErrorWithDetail(app *newrelic.Application, er error, client string, terminal string) {
	if er != nil {
		app.RecordCustomEvent("ERROR", map[string]interface{}{
			"Type":     "ERROR",
			"Detail":   er.Error(),
			"Client":   client,
			"Terminal": terminal,
		})
	}
}

func TrackNewRelic(msg string, app *newrelic.Application) (string, time.Time, *newrelic.Application) {
	return msg, time.Now(), app
}

func DurationNewRelic(msg string, start time.Time, app *newrelic.Application) {
	log.Printf("%v: %v\n", msg, time.Since(start))
	total := int(time.Since(start))
	t := (total * 1000)
	CaptureNewRelicMetric(app, msg, float64(t))
}
