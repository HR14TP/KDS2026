package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/glebarez/sqlite"
	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func IntToString(str int) string {
	return strconv.Itoa(str)
}

func showDebug(msg string) {
	if debug {
		log.Print(msg)
	}
}
func handleError(err error) {
	if err != nil {
		showDebug(err.Error())
		fmt.Println(err.Error())
		sentry.CaptureException(err)
	}
}
func getTimeSQL() string {
	t := time.Now()
	ts := t.Format("2006-01-02 15:04:05")
	return ts
}
func LoadConfiguration(file string) kdsmodels.KDSConfig {
	configFile, err := os.Open(file)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Println(err.Error())
	}
	defer configFile.Close()
	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)
	normalizeStandaloneMode(&config)
	return config
}

// normalizeStandaloneMode re-asserts the mutual exclusivity the Electron
// Client Config UI already enforces at save time (saveConfig/
// ApplyStandaloneMode in frontend.js: standalone forces passmode off and
// clears the assigned pass terminal) - but that's a UI-layer guarantee only,
// good for exactly as long as kdsconfig.json is never written any other
// way. If it's ever hand-edited, restored from a stale backup, or someday
// written by code that isn't saveConfig, nothing server-side stopped
// Standalonemode=true coexisting with a non-empty Passterminal/Passmode -
// which would leave isDeferredToPassTerminal (close_order.go) treating this
// screen as genuinely assigned to a pass terminal while this terminal's own
// frontend simultaneously behaves as standalone, scoped inconsistently
// against each other. Since every config read in this process funnels
// through LoadConfiguration (GetconfigHandler, announceOwnScreenPassAssignment,
// announceAsPassTerminal), this is the one place to re-assert it regardless
// of how the file on disk got into that state.
func normalizeStandaloneMode(c *kdsmodels.KDSConfig) {
	if c.Flags.Standalonemode {
		c.Flags.Passmode = false
		c.Client.Passterminal = ""
	}
}
func fixDateTime(column string) string {
	timeStamp, err := time.Parse("2006-01-02T15:04:05Z", column)
	if err != nil {
		sentry.CaptureException(err)
		return getTimeSQL()
	}
	return timeStamp.Format("2006-01-02 15:04:05")
}
func fixDateTime2(column *string) string {
	if column != nil {
		column2 := *column
		timeStamp, err := time.Parse("2006-01-02T15:04:05Z", column2)
		if err != nil {
			sentry.CaptureException(err)
			// do something with err...
			return getTimeSQL()
		}
		return timeStamp.Format("2006-01-02 15:04:05")
	} else {
		return "NULL"
	}
}
func dif2times(date1 string) []float64 {
	timeStamp, err := time.Parse("2006-01-02 15:04:05", date1)
	if err != nil {
		sentry.CaptureException(err)
		return []float64{0.0, 0.0}
	} else {
		date2 := time.Now()
		//spew.Dump(date1)
		date2 = date2.UTC()
		timeStamp = timeStamp.UTC()
		date2 = date2.Add(time.Hour * 2)
		//spew.Dump(date2)
		//spew.Dump(timeStamp)
		hs := date2.Sub(timeStamp).Hours()
		hs, mf := math.Modf(hs)
		ms := mf * 60
		ms, sf := math.Modf(ms)
		ss := sf * 60
		return []float64{ms, ss}
	}
}

func buildConfig(basepath string) (string, string, string) {
	base := strings.TrimSuffix(basepath, "\\")
	base = strings.TrimSuffix(base, "/")
	filebase := filepath.FromSlash(base + "/files")
	configbase := filepath.FromSlash(base + "/config/kdsconfig.json")
	dbbase := filepath.FromSlash(base + "/db/kds.db")
	return filebase, configbase, dbbase
}

func InitConfSQLite(file string, dbLocation string) *kdsmodels.KDSEnv {
	c := LoadConfiguration(file)
	env := &kdsmodels.KDSEnv{Config: &c}
	env.DB = createSQLITEDatabase(true, dbLocation)
	return env
}

func createSQLITEDatabase(debug bool, dbLocation string) *gorm.DB {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold: time.Second,   // Slow SQL threshold
			LogLevel:      logger.Silent, // Log level
			Colorful:      false,         // Disable color
		},
	)
	// kds-poller opens this same file independently (its own connection
	// pool, no shared lock) - without a busy_timeout, SQLite fails a write
	// immediately on lock contention instead of waiting for the other
	// process's writer to finish, which the mostly-unchecked ENV.DB.Save
	// call sites elsewhere silently swallow. A close order landing at the
	// same moment as a poller-driven ticket write could fail invisibly
	// and only "land" once some later, uncontended update touches the
	// order - looking exactly like a random delay.
	db, err := gorm.Open(sqlite.Open(dbLocation+"?_pragma=busy_timeout(5000)"), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	if err != nil {
		sentry.CaptureException(err)
		panic(err)
	}
	if debug {
		db.Config.Logger.LogMode(1)
		db.Config.Logger = newLogger
	}
	return db
}
