package kdsmodels

import "time"

type KDSTerminal struct {
	ID             int       `gorm:"column:id;primary_key" json:"id"`
	AccountID      int       `gorm:"column:account_id" json:"account_id"`
	Name           string    `gorm:"column:name" json:"name"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
	ConnectionCode string    `gorm:"column:connection_code" json:"connection_code"`
	HWID           string    `gorm:"column:hwid" json:"hwid"`
	ActiveConfig   string    `gorm:"-" json:"active_config"`
}

func (p *KDSTerminal) TableName() string {
	return "terminals"
}
