package ylamodels

import "time"

type YLAPage struct {
	ID         int       `gorm:"column:id;primary_key" json:"id"`
	IkentooBID int       `gorm:"column:ikentoo_b_id" json:"ikentoo_b_id"`
	Name       string    `gorm:"column:name" json:"name"`
	Type       string    `gorm:"column:type" json:"type"`
	Content    string    `gorm:"column:content" json:"content"`
	CreatedAt  time.Time `gorm:"column:created_at" json:"-"`
	UpdatedAt  time.Time `gorm:"column:updated_at" json:"-"`
}

// TableName sets the insert table name for this struct type
func (p *YLAPage) TableName() string {
	return "pages"
}
