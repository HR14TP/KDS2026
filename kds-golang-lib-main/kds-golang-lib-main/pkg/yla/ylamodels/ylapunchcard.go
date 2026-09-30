package ylamodels

import "time"

type YLAPunchCard struct {
	ID                        int       `gorm:"column:id;primary_key" json:"id"`
	UserID                    string    `gorm:"column:user_id" json:"user_id"`
	UUID                      string    `gorm:"column:uuid" json:"uuid"`
	IkentooBusinessLocationID string    `gorm:"column:ikentoo_business_location_id" json:"ikentoo_business_location_id"`
	IkentooBusinessID         string    `gorm:"column:ikentoo_business_id" json:"ikentoo_business_id"`
	CreatedAt                 time.Time `gorm:"column:created_at" json:"-"`
	UpdatedAt                 time.Time `gorm:"column:updated_at" json:"-"`
}

// TableName sets the insert table name for this struct type
func (p *YLAPunchCard) TableName() string {
	return "punch_cards"
}
