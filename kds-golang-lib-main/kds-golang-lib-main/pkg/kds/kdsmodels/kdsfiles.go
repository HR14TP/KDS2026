package kdsmodels

type KDSFiles struct {
	ID       int    `gorm:"column:id;primary_key" json:"id"`
	Hash     string `gorm:"column:hash" json:"hash"`
	Filename string `gorm:"column:filename" json:"filename"`
}

func (p *KDSFiles) TableName() string {
	return "kds_files"
}
