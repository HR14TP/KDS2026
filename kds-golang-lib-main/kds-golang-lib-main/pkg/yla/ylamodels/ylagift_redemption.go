package ylamodels

import (
	"time"
)

type YLAGiftRedemption struct {
	ID           int       `gorm:"column:id;primary_key" json:"id"`
	GiftID       int       `gorm:"column:gift_id" json:"gift_id"`
	UserID       string    `gorm:"column:user_id" json:"user_id"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at" json:"updated_at"`
	IkentooUUID  string    `gorm:"column:ikentoo_uuid" json:"ikentoo_uuid"`
	IkentooLuuid string    `gorm:"column:ikentoo_luuid" json:"ikentoo_luuid"`
	Item         YLAOffer  `gorm:"-" json:"item"`
}

// TableName sets the insert table name for this struct type
func (g *YLAGiftRedemption) TableName() string {
	return "gift_redemptions"
}
