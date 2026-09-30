package ylamodels

import "time"

type YLABusinessHour struct {
	ID        int       `gorm:"column:id;primary_key" json:"id"`
	BranchID  int       `gorm:"column:branch_id" json:"branch_id"`
	Day       int       `gorm:"column:day" json:"day"`
	OpenTime  string    `gorm:"column:open_time" json:"open_time"`
	CloseTime string    `gorm:"column:close_time" json:"close_time"`
	CreatedAt time.Time `gorm:"column:created_at" json:"-"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"-"`
}

// TableName sets the insert table name for this struct type
func (b *YLABusinessHour) TableName() string {
	return "business_hours"
}
