package ylamodels

import (
	"time"

	"gorm.io/gorm"
)

type YLAOffer struct {
	ID                int            `gorm:"column:id;primary_key" json:"id"`
	ItemID            int64          `gorm:"column:item_id" json:"item_id"`
	Type              string         `gorm:"column:type" json:"type"`
	Title             string         `gorm:"column:title" json:"title"`
	Description       *string        `gorm:"column:description" json:"description"`
	Price             float64        `gorm:"column:price" json:"price"`
	IkentooBusinessID string         `gorm:"column:ikentoo_business_id" json:"ikentoo_business_id"`
	CreatedAt         time.Time      `gorm:"column:created_at" json:"-"`
	UpdatedAt         time.Time      `gorm:"column:updated_at" json:"-"`
	Radius            int            `gorm:"column:radius" json:"radius"`
	DeletedAt         gorm.DeletedAt `gorm:"column:deleted_at" json:"-"`
	IkentooID         *string        `gorm:"column:ikentoo_id" json:"ikentoo_id"`
	File              YLAFile        `gorm:"-" json:"file"`
}

// TableName sets the insert table name for this struct type
func (o *YLAOffer) TableName() string {
	return "offers"
}

type YLAOfferSpending struct {
	ID                int       `gorm:"column:id;primary_key" json:"id"`
	ItemID            int64     `gorm:"column:item_id" json:"item_id"`
	Type              string    `gorm:"column:type" json:"type"`
	Title             string    `gorm:"column:title" json:"title"`
	Description       *string   `gorm:"column:description" json:"description"`
	Price             float64   `gorm:"column:price" json:"price"`
	IkentooBusinessID string    `gorm:"column:ikentoo_business_id" json:"ikentoo_business_id"`
	CreatedAt         time.Time `gorm:"column:created_at" json:"-"`
	UpdatedAt         time.Time `gorm:"column:updated_at" json:"-"`
	Radius            int       `gorm:"column:radius" json:"radius"`
	//DeletedAt         gorm.DeletedAt `gorm:"column:deleted_at" json:"-"`
	IkentooID *string `gorm:"column:ikentoo_id" json:"ikentoo_id"`
	File      YLAFile `gorm:"-" json:"file"`
}

// TableName sets the insert table name for this struct type
func (o *YLAOfferSpending) TableName() string {
	return "offers"
}
