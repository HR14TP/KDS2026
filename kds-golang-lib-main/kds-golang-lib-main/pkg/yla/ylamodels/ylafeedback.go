package ylamodels

import "time"

type YLAFeedback struct {
	ID        int       `gorm:"column:id;primary_key" json:"id"`
	Feedback  string    `gorm:"column:feedback" json:"feedback"`
	UserID    string    `gorm:"column:user_id" json:"user_id"`
	CompanyID string    `gorm:"column:company_id" json:"company_id"`
	Stars     int       `gorm:"column:stars" json:"stars"`
	CreatedAt time.Time `gorm:"column:created_at" json:"-"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"-"`
}

// TableName sets the insert table name for this struct type
func (f *YLAFeedback) TableName() string {
	return "feedback"
}
