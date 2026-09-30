package kdshandlers

import (
	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsactions"
	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
)

func GetDailyData(username string, password string, timeframe string) ([]kdsmodels.KDSApiDataAccount, error) {
	user, err := kdsactions.KDSLoginUser(username, password)
	if err != nil {
		return []kdsmodels.KDSApiDataAccount{}, err
	}
	data := kdsactions.KDSLoadApiData(user, timeframe)
	return data, err
}
