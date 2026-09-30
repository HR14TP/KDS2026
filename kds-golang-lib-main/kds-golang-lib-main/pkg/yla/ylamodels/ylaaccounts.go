package ylamodels

import "time"

// YLAAccount ... YLA Account
type YLAAccount struct {
	ID           int       `gorm:"column:id;primary_key" json:"id"`
	IkentooID    int       `gorm:"column:ikentoo_id" json:"ikentoo_id"`
	AuthUsername string    `gorm:"column:auth_username" json:"auth_username"`
	AuthPassword string    `gorm:"column:auth_password" json:"auth_password"`
	Theme        string    `gorm:"column:theme" json:"theme"`
	CompanyName  string    `gorm:"column:company_name" json:"company_name"`
	AdminName    string    `gorm:"column:admin_name" json:"admin_name"`
	AdminEmail   string    `gorm:"column:admin_email" json:"admin_email"`
	OnesignalID  string    `gorm:"column:onesignal_id" json:"onesignal_id"`
	OnesignalKey string    `gorm:"column:onesignal_key" json:"onesignal_key"`
	V3Token      string    `gorm:"column:v3token" json:"v3token"`
	UpdatedAt    time.Time `gorm:"column:updated_at" json:"updated_at"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
}

// TableName sets the insert table name for this struct type
func (a *YLAAccount) TableName() string {
	return "accounts"
}
