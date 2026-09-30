package ylamodels

import "time"

type YLAMembership struct {
	ID                int       `gorm:"column:id;primary_key" json:"id"`
	Name              string    `gorm:"column:name" json:"name"`
	Type              string    `gorm:"column:type" json:"type"`
	IkentooBusinessID int       `gorm:"column:ikentoo_business_id" json:"ikentoo_business_id"`
	CreatedAt         time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// TableName sets the insert table name for this struct type
func (m *YLAMembership) TableName() string {
	return "memberships"
}
