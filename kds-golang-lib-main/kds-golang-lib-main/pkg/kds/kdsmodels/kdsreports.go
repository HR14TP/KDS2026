package kdsmodels

type KDSReports struct {
	ID     int    `gorm:"column:id;primary_key" json:"id"`
	UserID int    `gorm:"column:user_id" json:"user_id"`
	Name   string `gorm:"column:name" json:"name"`
	Desc   string `gorm:"column:desc" json:"desc"`
}

func (p *KDSReports) TableName() string {
	return "reports"
}
