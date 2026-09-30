package ylamodels

import "time"

type YLAEmail struct {
	ID        int       `gorm:"column:id;primary_key" json:"id"`
	OrderID   int       `gorm:"column:order_id" json:"order_id"`
	ToEmail   string    `gorm:"column:toemail" json:"toemail"`
	Args      string    `gorm:"column:args" json:"args"`
	Status    int       `gorm:"column:status" json:"status"`
	Template  string    `gorm:"column:template" json:"template"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// TableName sets the insert table name for this struct type
func (e *YLAEmail) TableName() string {
	return "emails"
}
