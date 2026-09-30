package ylamodels

import "time"

type YLAIkentooMessage struct {
	ID                 int       `gorm:"column:id;primary_key" json:"id"`
	Message            string    `gorm:"column:message" json:"message"`
	Origin             string    `gorm:"column:origin" json:"origin"`
	OrderID            string    `gorm:"column:order_id" json:"order_id"`
	AccountID          string    `gorm:"column:account_id" json:"account_id"`
	BusinessLocationID string    `gorm:"column:business_location_id" json:"business_location_id"`
	AccountIdentifier  string    `gorm:"column:account_identifier" json:"account_identifier"`
	Status             string    `gorm:"column:status" json:"status"`
	CreatedAt          time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// TableName sets the insert table name for this struct type
func (p *YLAIkentooMessage) TableName() string {
	return "ikentoo_messages"
}
