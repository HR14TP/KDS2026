package ylamodels

import "time"

type PayfastAdhoc struct {
	ID          int       `gorm:"column:id;primary_key" json:"id"`
	MerchantID  string    `gorm:"column:merchant_id" json:"merchant_id"`
	Token       string    `gorm:"column:token" json:"token"`
	Signature   string    `gorm:"column:signature" json:"signature"`
	Description string    `gorm:"column:description" json:"description"`
	Amount      int       `gorm:"column:amount" json:"amount"`
	Response    string    `gorm:"column:response" json:"response"`
	CreatedAt   time.Time `gorm:"-" json:"-"`
	UpdatedAt   time.Time `gorm:"-" json:"-"`
}

func (m *PayfastAdhoc) TableName() string {
	return "payfast_adhoc"
}

type YLAUserCard struct {
	ID        int       `gorm:"column:id;primary_key" json:"id"`
	UserID    int       `gorm:"column:user_id" json:"user_id"`
	Token     string    `gorm:"column:token" json:"token"`
	Tag       string    `gorm:"column:tag" json:"tag"`
	DeviceID  string    `gorm:"column:device_id" json:"device_id"`
	BranchID  int       `gorm:"column:branch_id" json:"branch_id"`
	CreatedAt time.Time `gorm:"column:created_at" json:"-"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"-"`
}

// TableName sets the insert table name for this struct type
func (s *YLAUserCard) TableName() string {
	return "user_cards"
}
