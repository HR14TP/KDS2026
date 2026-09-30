package ylamodels

import (
	"time"
)

type YLASequence struct {
	ID                   int                `gorm:"column:id;primary_key" json:"id"`
	Name                 string             `gorm:"column:name" json:"name"`
	Type                 string             `gorm:"column:type" json:"type"`
	Order                int                `gorm:"column:order" json:"order"`
	MustSelect           int                `gorm:"column:mustselect" json:"mustselect"`
	Minimum              int                `gorm:"column:minimum" json:"minimum"`
	Maximum              int                `gorm:"column:maximum" json:"maximum"`
	Reflectprice         int                `gorm:"column:reflectprice" json:"reflectprice"`
	Itemdiscountpositive int                `gorm:"column:itemdiscountpositive" json:"itemdiscountpositive"`
	Itemdiscountnegative int                `gorm:"column:itemdiscountnegative" json:"itemdiscountnegative"`
	IkentooBusinessID    string             `gorm:"column:ikentoo_business_id" json:"ikentoo_business_id"`
	DeletedAt            time.Time          `gorm:"column:deleted_at" json:"-"`
	CreatedAt            time.Time          `gorm:"column:created_at" json:"-"`
	UpdatedAt            time.Time          `gorm:"column:updated_at" json:"-"`
	SequenceValues       []YLASequenceValue `gorm:"foreignkey:SequenceID"`
	SingleSelected       string             `gorm:"-" json:"singleselected"`
}

// TableName sets the insert table name for this struct type
func (s *YLASequence) TableName() string {
	return "sequences"
}
