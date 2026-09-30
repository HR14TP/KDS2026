package ylamodels

import (
	"time"

	"gorm.io/gorm"
)

type YLAItem struct {
	ID                        int                        `gorm:"column:id;primary_key" json:"id"`
	IkentooID                 string                     `gorm:"column:ikentoo_id" json:"ikentoo_id"`
	Producible                int                        `gorm:"column:producible" json:"producible"`
	Name                      string                     `gorm:"column:name" json:"name"`
	PhotoID                   string                     `gorm:"column:photo_id" json:"photo_id"`
	Description               string                     `gorm:"column:description" json:"description"`
	DeletedAt                 gorm.DeletedAt             `gorm:"column:deleted_at" json:"-"`
	CreatedAt                 time.Time                  `gorm:"column:created_at" json:"-"`
	UpdatedAt                 time.Time                  `gorm:"column:updated_at" json:"-"`
	Sku                       int                        `gorm:"column:sku" json:"sku"`
	BarCode                   string                     `gorm:"column:bar_code" json:"bar_code"`
	StatisticGroup            string                     `gorm:"column:statistic_group" json:"statistic_group"`
	CostPrice                 float64                    `gorm:"column:cost_price" json:"cost_price"`
	IkentooBusinessLocationID string                     `gorm:"column:ikentoo_business_location_id" json:"ikentoo_business_location_id"`
	IkentooBusinessID         string                     `gorm:"column:ikentoo_business_id" json:"ikentoo_business_id"`
	Disabled                  int                        `gorm:"column:disabled" json:"disabled"`
	PreparationTime           int                        `gorm:"column:preparation_time" json:"preparation_time"`
	IkentooSKU                string                     `gorm:"column:ikentoo_sku" json:"ikentoo_sku"`
	File                      YLAFile                    `gorm:"-" json:"file"`
	Sequences                 []YLASequence              `gorm:"-"`
	ProductionInstructions    []YLAProductionInstruction `gorm:"-"`
}

// TableName sets the insert table name for this struct type
func (i *YLAItem) TableName() string {
	return "items"
}
