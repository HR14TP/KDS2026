package ylamodels

import "time"

type YLAMenu struct {
	ID                        int       `gorm:"column:id;primary_key" json:"id"`
	ParentScreenID            *int      `gorm:"column:parent_screen_id" json:"parent_screen_id"`
	Name                      string    `gorm:"column:name" json:"name"`
	IkentooID                 string    `gorm:"column:ikentoo_id" json:"ikentoo_id"`
	IkentooBusinessLocationID string    `gorm:"column:ikentoo_business_location_id" json:"ikentoo_business_location_id"`
	IkentooBusinessID         string    `gorm:"column:ikentoo_business_id" json:"ikentoo_business_id"`
	DeletedAt                 time.Time `gorm:"column:deleted_at" json:"-"`
	CreatedAt                 time.Time `gorm:"column:created_at" json:"-"`
	UpdatedAt                 time.Time `gorm:"column:updated_at" json:"-"`
	OrderNo                   int       `gorm:"column:order_no" json:"order_no"`
	Menus                     []YLAMenu `gorm:"foreignkey:ID;references:ParentScreenID"`
	Items                     []YLAItem `gorm:"-"`
	Img                       string    `gorm:"column:img" json:"img"`
	Icon                      string    `gorm:"column:icon" json:"icon"`
}

// TableName sets the insert table name for this struct type
func (m *YLAMenu) TableName() string {
	return "menus"
}
