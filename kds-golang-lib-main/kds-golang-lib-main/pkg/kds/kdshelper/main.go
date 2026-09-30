package kdshelper

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
)

func BuildConfig(basepath string) (string, string, string) {
	base := strings.TrimSuffix(basepath, "\\")
	base = strings.TrimSuffix(base, "/")
	filebase := filepath.FromSlash(base + "/files")
	configbase := filepath.FromSlash(base + "/config/kdsconfig.json")
	dbbase := filepath.FromSlash(base + "/db/kds.db")
	return filebase, configbase, dbbase
}

func LoadConfiguration(file string) kdsmodels.KDSConfig {
	var config kdsmodels.KDSConfig
	configFile, _ := os.Open(file)
	defer configFile.Close()
	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)
	return config
}

func LoadConfigurationNew(file string) kdsmodels.KDSConfigNew {
	var config kdsmodels.KDSConfigNew
	configFile, _ := os.Open(file)
	defer configFile.Close()
	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)
	return config
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
