package kdshelper

import (
	"fmt"
	"os"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/reverent911/kds-golang-lib/pkg/generic/errors"
	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
)

var KDSTCPSERVER_SENTRY_DSN = "https://f01080ff4e55453f9e207d2fe3471c86@o890609.ingest.sentry.io/5872562"
var KDSPOLLER_SENTRY_DSN = "https://7bef2acc7975489ab11c50989c8fdf72@o890609.ingest.sentry.io/5872804"
var KDSWEBSERVER_SENTRY_DSN = "https://806452232567418fb85ab037f9a627d4@o890609.ingest.sentry.io/5874058"
var KDSAPP_SENTRY_DSN = "https://619e2735043b41288fa142974b06b541@o157636.ingest.sentry.io/1760939"
var KDS_WORKER_SENTRY_DSN = "https://e973d46a58954ba9b7b08e78c5637b4f@o890609.ingest.sentry.io/6231771"

var KDSTCPSERVER_NR = "KDSTCPSERVER"
var KDSPOLLER_NR = "KDSPOLLER"
var KDSWEBSERVER_NR = "KDS Webserver"
var KDSAPP_NR = ""
var KDSNEWRELIC = os.Getenv("KDS_NEWRELIC_LICENSE_KEY")

type Services struct {
	KDSTCPSERVER string
	KDSPOLLER    string
	KDSWEBSERVER string
	KDSAPP       string
	NEWRELICKEY  string
	KDSWORKER    string
}

func GetSentryServices() *Services {
	return &Services{
		KDSTCPSERVER: KDSTCPSERVER_SENTRY_DSN,
		KDSPOLLER:    KDSPOLLER_SENTRY_DSN,
		KDSWEBSERVER: KDSWEBSERVER_SENTRY_DSN,
		KDSAPP:       KDSAPP_SENTRY_DSN,
		KDSWORKER:    KDS_WORKER_SENTRY_DSN,
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

func HandelGlobalErrorTXN(err error, sentryservice string, newrelicserver string, txn *newrelic.Transaction) {
	if err != nil {
		errors.CaptureSentryError(sentryservice, err)
		txn.NoticeError(err)
		fmt.Println(err.Error())
	}
}
func HandelGlobalErrorAPP(err error, sentryservice string, newrelicserver string, app *newrelic.Application) {
	if err != nil {
		errors.CaptureSentryError(sentryservice, err)
		errors.CaptureNewRelicError(app, err)
		fmt.Println(err.Error())
	}
}
func HandelGlobalError(err error, sentryservice string) {
	if err != nil {
		errors.CaptureSentryError(sentryservice, err)
		fmt.Println(err.Error())
	}
}

func HandelGlobalErrorConfig(err error, sentryservice string, config kdsmodels.KDSConfigNew) {
	if err != nil {
		errors.CaptureKDSErrorWithConfig(sentryservice, err, config)
		fmt.Println(err.Error())
	}
}
