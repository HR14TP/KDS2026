package kdsmodels

import "time"

type KDSSubitems struct {
	ID        int     `gorm:"column:id;primary_key" json:"id"`
	Qty       int     `gorm:"column:qty" json:"qty"`
	Name      string  `gorm:"column:name" json:"name"`
	Status    string  `gorm:"column:status" json:"status"`
	KDSItemID int     `gorm:"column:kds_item_id" json:"kds_item_id"`
	QtyRaw    float64 `gorm:"column:qty_raw" json:"qty_raw"`
}

func (p *KDSSubitems) TableName() string {
	return "kds_subitems"
}

type KDSSubitemsMySQL struct {
	ID         int       `gorm:"column:id;primary_key" json:"id"`
	LocalID    int       `gorm:"column:local_id;primary_key" json:"local_id"`
	TerminalID int       `gorm:"column:terminal_id" json:"terminal_id"`
	Qty        int       `gorm:"column:qty" json:"qty"`
	Name       string    `gorm:"column:name" json:"name"`
	Status     string    `gorm:"column:status" json:"status"`
	KDSItemID  int       `gorm:"column:kds_item_id" json:"kds_item_id"`
	CreatedAt  time.Time `gorm:"-" json:"created_at"`
}

func (p *KDSSubitemsMySQL) TableName() string {
	return "kds_subitems"
}
