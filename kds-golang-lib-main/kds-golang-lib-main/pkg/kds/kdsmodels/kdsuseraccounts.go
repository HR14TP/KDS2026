package kdsmodels

type KDSUserAccounts struct {
	ID        int `gorm:"column:id;primary_key" json:"id"`
	UserID    int `gorm:"column:user_id" json:"user_id"`
	AccountID int `gorm:"column:account_id" json:"account_id"`
}

func (p *KDSUserAccounts) TableName() string {
	return "user_accounts"
}
