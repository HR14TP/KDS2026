package database

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/glebarez/sqlite"
	"github.com/reverent911/kds-golang-lib/pkg/generic/helper"
	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var debug = false
var NewLogger = logger.New(
	log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
	logger.Config{
		SlowThreshold: time.Second, // Slow SQL threshold
		LogLevel:      logger.Info, // Log level
		Colorful:      true,        // Disable color
	},
)

func DOMYSQLConnection(dsn string) *gorm.DB {
	if debug {
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: NewLogger})
		if err != nil {
			panic(err)
		}
		return db
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	return db
}

func DOMYSQLDebugConnection(dsn string) *gorm.DB {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: NewLogger})
	if err != nil {
		panic(err)
	}
	return db
}

func YLADebuggedConnection() *gorm.DB {
	fmt.Println("CREATING YLA CONNECTION===============================================")
	dsn := "tabit:tamarisk2@tcp(develop-database.cehyymhejbdm.eu-west-1.rds.amazonaws.com:3306)/yla?charset=utf8mb4&parseTime=True&loc=Local"
	db := DOMYSQLDebugConnection(dsn)
	sqlDB, err := db.DB()
	if err != nil {
		fmt.Println(err)
	}
	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDB.SetMaxIdleConns(10)

	// SetMaxOpenConns sets the maximum number of open connections to the database.
	sqlDB.SetMaxOpenConns(50)

	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	return db
}

func YLALambdaConnection() *gorm.DB {
	fmt.Println("CREATING YLA CONNECTION===============================================")
	dsn := "tabit:tamarisk2@tcp(develop-database.cehyymhejbdm.eu-west-1.rds.amazonaws.com:3306)/yla?charset=utf8mb4&parseTime=True&loc=Local"
	db := DOMYSQLConnection(dsn)
	// Get generic database object sql.DB to use its functions
	sqlDB, err := db.DB()
	if err != nil {
		fmt.Println(err)
	}
	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDB.SetMaxIdleConns(5)

	// SetMaxOpenConns sets the maximum number of open connections to the database.
	sqlDB.SetMaxOpenConns(10)

	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	return db
}

func YLAConnection() *gorm.DB {
	fmt.Println("CREATING YLA CONNECTION===============================================")
	dsn := "tabit:tamarisk2@tcp(develop-database.cehyymhejbdm.eu-west-1.rds.amazonaws.com:3306)/yla?charset=utf8mb4&parseTime=True&loc=Local"
	return DOMYSQLConnection(dsn)
}
func KDSConnection() *gorm.DB {
	fmt.Println("CREATING KDS CONNECTION===============================================")
	dsn := "tabit:tamarisk2@tcp(develop-database.cehyymhejbdm.eu-west-1.rds.amazonaws.com:3306)/kds?charset=utf8mb4&parseTime=True&loc=Africa%2FHarare"
	db := DOMYSQLDebugConnection(dsn)
	// Get generic database object sql.DB to use its functions
	sqlDB, err := db.DB()
	if err != nil {
		fmt.Println(err)
	}
	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDB.SetMaxIdleConns(10)

	// SetMaxOpenConns sets the maximum number of open connections to the database.
	sqlDB.SetMaxOpenConns(20)

	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	return db
}

// ContactlessConnection ... Create Contactless Database Connection
func ContactlessConnection() *gorm.DB {
	fmt.Println("CREATING CONTACTLESS CONNECTION===============================================")
	dsn := "tabit:tamarisk2@tcp(develop-database.cehyymhejbdm.eu-west-1.rds.amazonaws.com:3306)/contactless?charset=utf8mb4&parseTime=True&loc=Local"
	return DOMYSQLConnection(dsn)
}

func FIGEROConnection() *gorm.DB {
	fmt.Println("CREATING FIGERO CONNECTION===============================================")
	//dsn := "admin:Tabletpos1@tcp(figerodb.ckztw33aipue.eu-west-1.rds.amazonaws.com:3306)/figero?charset=utf8mb4&parseTime=True&loc=Local"
	dsn := "tabit:tamarisk2@tcp(develop-database.cehyymhejbdm.eu-west-1.rds.amazonaws.com:3306)/figero?charset=utf8mb4&parseTime=True&loc=Local"

	db := DOMYSQLDebugConnection(dsn)
	// Get generic database object sql.DB to use its functions
	sqlDB, err := db.DB()
	if err != nil {
		fmt.Println(err)
	}
	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDB.SetMaxIdleConns(10)

	// SetMaxOpenConns sets the maximum number of open connections to the database.
	sqlDB.SetMaxOpenConns(20)

	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	return db
}

func CloseDatabase(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		log.Println(err)
	} else {
		defer sqlDB.Close()
	}
}

func InitConf(file string) *kdsmodels.KDSEnv {
	c := loadConfiguration(file)
	env := &kdsmodels.KDSEnv{Config: c}
	env.DB = createDatabase(env.Config, false)
	return env
}

func InitConfSQLite(file string, dbLocation string) *kdsmodels.KDSEnv {
	c := loadConfiguration(file)
	env := &kdsmodels.KDSEnv{Config: c}
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
	db, err := gorm.Open(sqlite.Open(dbLocation), &gorm.Config{
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

func createDatabase(conf *kdsmodels.KDSConfig, debug bool) *gorm.DB {
	db, err := gorm.Open(mysql.Open(
		conf.Database.Username+":"+conf.Database.Password+"@tcp("+conf.Database.Host+":"+helper.IntToString(conf.Database.Port)+")/"+conf.Database.Schema+"?parseTime=true"), &gorm.Config{})
	if err != nil {
		sentry.CaptureException(err)
		panic(err)
	}
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold: time.Second,   // Slow SQL threshold
			LogLevel:      logger.Silent, // Log level
			Colorful:      false,         // Disable color
		},
	)
	if debug {
		db.Config.Logger.LogMode(1)
		db.Config.Logger = newLogger
	}
	return db
}

var config *kdsmodels.KDSConfig

func loadConfiguration(file string) *kdsmodels.KDSConfig {
	configFile, err := os.Open(file)
	defer configFile.Close()
	if err != nil {
		fmt.Println(err.Error())
	}
	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)
	return config
}
