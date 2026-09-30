package ylamodels

type YLAClientConfig struct {
	ID             int    `gorm:"column:id;primary_key" json:"id"`
	ClientID       int    `gorm:"column:client_id" json:"client_id"`
	ClientConfig   string `gorm:"column:client_config" json:"client_config"`
	Version        string `gorm:"column:version" json:"version"`
	UpdateRequired bool   `gorm:"column:update_required" json:"update_required"`
}

// TableName sets the insert table name for this struct type
func (m *YLAClientConfig) TableName() string {
	return "appconfig"
}
