package ylamodels

import "time"

type YLAUserAddress struct {
	ID           int       `gorm:"column:id;primary_key" json:"id"`
	UserID       int       `gorm:"column:user_id" json:"user_id"`
	Address      string    `gorm:"column:address" json:"address"`
	Town         string    `gorm:"column:town" json:"town"`
	Postal       string    `gorm:"column:postal" json:"postal"`
	Tag          string    `gorm:"column:tag" json:"tag"`
	BuildingName string    `gorm:"column:building_name" json:"building_name"`
	StreetNumber int       `gorm:"column:street_number" json:"street_number"`
	StreetName   string    `gorm:"column:street_name" json:"street_name"`
	Floor        int       `gorm:"column:floor" json:"floor"`
	Valid        int       `gorm:"column:valid" json:"valid"`
	Lat          string    `gorm:"column:lat" json:"lat"`
	Lng          string    `gorm:"column:lng" json:"lng"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"-"`
	UpdatedAt    time.Time `gorm:"column:updated_at" json:"-"`
}

// TableName sets the insert table name for this struct type
func (s *YLAUserAddress) TableName() string {
	return "user_address"
}
