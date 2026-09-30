package errors

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
)

var KDSTCPSERVER_SENTRY_DSN = "https://f01080ff4e55453f9e207d2fe3471c86@o890609.ingest.sentry.io/5872562"
var KDSPOLLER_SENTRY_DSN = "https://7bef2acc7975489ab11c50989c8fdf72@o890609.ingest.sentry.io/5872804"
var KDSWEBSERVER_SENTRY_DSN = "https://806452232567418fb85ab037f9a627d4@o890609.ingest.sentry.io/5874058"
var KDSAPP_SENTRY_DSN = "https://619e2735043b41288fa142974b06b541@o157636.ingest.sentry.io/1760939"
var KDS_WORKER_SENTRY_DSN = "https://e973d46a58954ba9b7b08e78c5637b4f@o890609.ingest.sentry.io/6231771"
var CONTACTLESS_SYNC_DSN = "https://eccf030b3e5f43288a3be4445e202aa5@o890609.ingest.sentry.io/6236851"
var IKENTOO_API_DSN = "https://cae2096f8a65419b8ee35d256bf219d3@o890609.ingest.sentry.io/6236868"

var KDSTCPSERVER_NR = "KDSTCPSERVER"
var KDSPOLLER_NR = "KDSPOLLER"
var KDSWEBSERVER_NR = "KDS Webserver"
var KDSAPP_NR = ""
var KDSNEWRELIC = os.Getenv("KDS_NEWRELIC_LICENSE_KEY")

type Services struct {
	KDSTCPSERVER     string
	KDSPOLLER        string
	KDSWEBSERVER     string
	KDSAPP           string
	NEWRELICKEY      string
	KDSWORKER        string
	CONTACTLESS_SYNC string
	IKENTOO_API      string
}

func GetSentryServices() *Services {
	return &Services{
		KDSTCPSERVER:     KDSTCPSERVER_SENTRY_DSN,
		KDSPOLLER:        KDSPOLLER_SENTRY_DSN,
		KDSWEBSERVER:     KDSWEBSERVER_SENTRY_DSN,
		KDSAPP:           KDSAPP_SENTRY_DSN,
		KDSWORKER:        KDS_WORKER_SENTRY_DSN,
		CONTACTLESS_SYNC: CONTACTLESS_SYNC_DSN,
		IKENTOO_API:      IKENTOO_API_DSN,
	}
}

func GetNewRelicServices() *Services {
	return &Services{
		KDSTCPSERVER: KDSTCPSERVER_NR,
		KDSPOLLER:    KDSPOLLER_NR,
		KDSWEBSERVER: KDSWEBSERVER_NR,
		KDSAPP:       KDSAPP_NR,
		NEWRELICKEY:  KDSNEWRELIC,
	}
}

func CaptureNewRelicError(app *newrelic.Application, er error) {
	if er != nil {
		app.RecordCustomEvent("ERROR", map[string]interface{}{
			"Type":   "ERROR",
			"Detail": er.Error(),
		})
	}
}

func CaptureSentryError(service string, sentryerror error) {
	//Init Sentry
	err := sentry.Init(sentry.ClientOptions{
		Dsn: service,
	})
	//New Relic The Error
	if err != nil {
		log.Printf("Unable To Initialize Sentry %s", err)
	}
	// Flush buffered events before the program terminates.
	defer sentry.Flush(2 * time.Second)
	//Capture Sentry Error
	if sentryerror != nil {
		sentry.CaptureException(sentryerror)
	}
}

func CaptureKDSErrorWithConfig(service string, sentryerror error, config kdsmodels.KDSConfigNew) {
	//Init Sentry
	err := sentry.Init(sentry.ClientOptions{
		Dsn:         service,
		ServerName:  config.Client.Name,
		Dist:        config.Client.Terminal,
		Environment: config.Client.Serial,
		Release:     service,
	})
	//New Relic The Error
	if err != nil {
		log.Printf("Unable To Initialize Sentry %s", err)
	}
	// Flush buffered events before the program terminates.
	defer sentry.Flush(2 * time.Second)
	//Capture Sentry Error
	if sentryerror != nil {
		sentry.CaptureException(sentryerror)
	}
}

// CaptureYlaSentry ... Capture YLA Sentry Error
func YLASentryCapture(sentryerror error) {
	if sentryerror != nil {
		err := sentry.Init(sentry.ClientOptions{
			Dsn: "https://179e4b2334ba4b1f93eed9f0d079c24e@o157636.ingest.sentry.io/1763448",
		})
		if err != nil {
			log.Fatalf("sentry.Init: %s", err)
		}
		// Flush buffered events before the program terminates.
		defer sentry.Flush(2 * time.Second)
		sentry.CaptureException(sentryerror)
	}
}
func YLANotifyOnError(er error) {
	if er != nil {
		YLASentryCapture(er)
	}
}

func CreateNewRelic(appname string) (*newrelic.Application, error) {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName(appname),
		newrelic.ConfigLicense(GetNewRelicServices().NEWRELICKEY),
		newrelic.ConfigDistributedTracerEnabled(true),
		newrelic.ConfigEnabled(true),
	)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	err = app.WaitForConnection(10 * time.Second)
	if err != nil {
		return nil, err
	}
	return app, nil
}
