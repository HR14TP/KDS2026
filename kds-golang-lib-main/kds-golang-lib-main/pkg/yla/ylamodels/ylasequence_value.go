package ylamodels

import (
	"time"
)

type YLASequenceValue struct {
	ID                int       `gorm:"column:id;primary_key" json:"id"`
	SequenceID        int       `gorm:"column:sequence_id" json:"sequence_id"`
	Name              string    `gorm:"column:name" json:"name"`
	Price             float64   `gorm:"column:price" json:"price"`
	Order             int       `gorm:"column:order" json:"order"`
	DeletedAt         time.Time `gorm:"column:deleted_at" json:"-"`
	CreatedAt         time.Time `gorm:"column:created_at" json:"-"`
	UpdatedAt         time.Time `gorm:"column:updated_at" json:"-"`
	Sku               int       `gorm:"column:sku" json:"sku"`
	Chosen            bool      `gorm:"-" json:"chosen"`
	IkentooBusinessID *string   `gorm:"column:ikentoo_business_id" json:"ikentoo_business_id"`
}

// TableName sets the insert table name for this struct type
func (s *YLASequenceValue) TableName() string {
	return "sequence_values"
}
