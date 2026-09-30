package ylamodels

import "time"

type YLARequests struct {
	Input        string    `gorm:"column:input" json:"input"`
	URL          string    `gorm:"column:url" json:"url"`
	Status       int       `gorm:"column:status" json:"status"`
	Response     string    `gorm:"column:response" json:"response"`
	AuthUser     string    `gorm:"column:auth_user" json:"auth_user"`
	AuthPassword string    `gorm:"column:auth_password" json:"auth_password"`
	AuthToken    string    `gorm:"column:auth_token" json:"auth_token"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// TableName sets the insert table name for this struct type
func (p *YLARequests) TableName() string {
	return "requests"
}
