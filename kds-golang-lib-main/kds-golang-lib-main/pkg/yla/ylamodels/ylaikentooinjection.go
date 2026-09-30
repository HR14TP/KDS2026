package ylamodels

type IkentooInjection struct {
	ID        int    `gorm:"column:id;primary_key" json:"id"`
	Theorder  string `gorm:"column:theorder" json:"theorder"`
	Injection string `gorm:"column:injection" json:"injection"`
	Response  string `gorm:"column:response" json:"response"`
}

// TableName sets the insert table name for this struct type
func (p *IkentooInjection) TableName() string {
	return "ikentoo_injections"
}
