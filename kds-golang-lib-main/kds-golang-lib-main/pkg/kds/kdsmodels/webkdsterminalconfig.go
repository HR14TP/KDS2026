package kdsmodels

import "time"

type KDSTerminalConfig struct {
	ID         int       `gorm:"column:id;primary_key" json:"id"`
	TerminalID int       `gorm:"column:terminal_id" json:"terminal_id"`
	Config     string    `gorm:"column:config" json:"config"`
	LastHWID   string    `gorm:"column:last_hwid" json:"last_hwid"`
	CreatedAt  time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at" json:"updated_at"`
	SyncedAt   time.Time `gorm:"column:synced_at" json:"synced_at"`
}

func (p *KDSTerminalConfig) TableName() string {
	return "terminal_configs"
}
