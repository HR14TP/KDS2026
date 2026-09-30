package ylamodels

import "time"

type YLACompany struct {
	ID           uint        `gorm:"column:id;primary_key" json:"id"`
	Name         string      `gorm:"column:name" json:"name"`
	PhotoID      int         `gorm:"column:photo_id" json:"photo_id"`
	ColourScheme string      `gorm:"column:colour_scheme" json:"colour_scheme"`
	Address      string      `gorm:"column:address" json:"address"`
	Location     string      `gorm:"column:location" json:"location"`
	SubDomainURL string      `gorm:"column:sub_domain_url" json:"sub_domain_url"`
	FacebookID   string      `gorm:"column:facebook_id" json:"facebook_id"`
	TwitterID    string      `gorm:"column:twitter_id" json:"twitter_id"`
	InstagramID  string      `gorm:"column:instagram_id" json:"instagram_id"`
	Active       int         `gorm:"column:active" json:"active"`
	DeletedAt    time.Time   `gorm:"column:deleted_at" json:"-"`
	CreatedAt    time.Time   `gorm:"column:created_at" json:"-"`
	UpdatedAt    time.Time   `gorm:"column:updated_at" json:"-"`
	IkentooID    int64       `gorm:"column:ikentoo_id" json:"ikentoo_id"`
	Branches     []YLABranch `gorm:"foreignkey:CompanyID"`
	Pages        []YLAPage   `gorm:"-"`
}

// TableName sets the insert table name for this struct type
func (c *YLACompany) TableName() string {
	return "companies"
}
