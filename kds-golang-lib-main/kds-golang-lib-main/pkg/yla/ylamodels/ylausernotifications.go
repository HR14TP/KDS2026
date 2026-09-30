package ylamodels

import "time"

type YLAUserNotification struct {
	ID        int       `gorm:"column:id;primary_key" json:"id"`
	UserID    int       `gorm:"column:user_id" json:"user_id"`
	Message   string    `gorm:"column:message" json:"message"`
	IkentooID int       `gorm:"column:ikentoo_id" json:"ikentoo_id"`
	Response  string    `gorm:"column:response" json:"response"`
	CreatedAt time.Time `gorm:"column:created_at" json:"-"`
}

// TableName sets the insert table name for this struct type
func (s *YLAUserNotification) TableName() string {
	return "user_notifications"
}
