package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/davecgh/go-spew/spew"
	"github.com/getsentry/sentry-go"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
)

var debug = true

func repNums(s string) string {

	if !KDSConfig.Flags.EnableGDPR {
		log.Print("GDPR Disabled")
		return s
	}

	out := make([]rune, len(s)) // len(s) is bytes not runes, this is just estimation

	i, added := 0, false
	for _, r := range s {
		if r >= '0' && r <= '9' {
			if added {
				continue
			}
			added, out[i] = true, '#'
		} else {
			added, out[i] = false, r
		}
		i++
	}
	return string(out[:i])
}

// ShowDebug ... ShowDebug
func ShowDebug(msg string) {
	if debug {
		log.Print(msg)
	}
}

// ShowDebugInt ... ShowDebugInt
func ShowDebugInt(msg interface{}) {
	if debug {
		spew.Dump(msg)
	}
}

func UtilStringToInt(str string) int {
	i, err := strconv.Atoi(str)
	if err != nil {
		i = 1
	}
	return i
}

// HandleError ... HandleError
func HandleError(err error, txn *newrelic.Transaction) {
	if err != nil {
		if txn != nil {
			txn.NoticeError(err)
		}
		ShowDebug(err.Error())
		//sentry.CaptureException(err)
		os.Exit(1)
	}
}

// HandleErrorNonFatal ... HandleErrorNonFatal
func HandleErrorNonFatal(err error, txn *newrelic.Transaction) {
	if err != nil {
		if txn != nil {
			txn.NoticeError(err)
		}
		ShowDebug(err.Error())
		//sentry.CaptureException(err)
	}
}

// HandleErrorFatal ... HandleErrorFatal
func HandleErrorFatal(err error) {
	if err != nil {
		log.Print(err)
		ShowDebug(err.Error())
		sentry.CaptureException(err)
		os.Exit(1)
	}
}

// LoadConfiguration ... LoadConfiguration
func LoadConfiguration(file string) kdsmodels.KDSConfig {
	var config kdsmodels.KDSConfig
	configFile, _ := os.Open(file)
	//myerror.HandelGlobalError(err, myerror.GetSentryServices().KDSPOLLER)
	defer configFile.Close()
	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)
	return config
}

func UtilIntToString(str int) string {
	return strconv.Itoa(str)
}

// NotifyServer ... NotifyServer
func NotifyServer(config kdsmodels.KDSConfig) {
	client := http.Client{
		Timeout: time.Duration(10 * time.Second),
	}
	url := "http://" + config.Webserver.Host + ":" + UtilIntToString(config.Webserver.Port) + "/updated"
	_, err := client.Get(url)
	HandleErrorNonFatal(err, nil)
}

// GenHash ... GenHash
func GenHash(f *os.File) string {
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func buildConfig(basepath string) (string, string, string) {
	base := strings.TrimSuffix(basepath, "\\")
	base = strings.TrimSuffix(base, "/")
	filebase := filepath.FromSlash(base + "/files")
	configbase := filepath.FromSlash(base + "/config/kdsconfig.json")
	dbbase := filepath.FromSlash(base + "/db/kds.db")
	return filebase, configbase, dbbase
}

// RemoveNonUTF ... RemoveNonUTF
func RemoveNonUTF(s string) string {
	if !utf8.ValidString(s) {
		v := make([]rune, 0, len(s))
		for i, r := range s {
			if r == utf8.RuneError {
				_, size := utf8.DecodeRuneInString(s[i:])
				if size == 1 {
					continue
				}
			}
			v = append(v, r)
		}
		s = string(v)
	}
	return s
}
