package ylamodels

type YLABranchPaymentType struct {
	BranchID      int64 `gorm:"column:branch_id;primary_key" json:"branch_id"`
	PaymentTypeID int64 `gorm:"column:payment_type_id" json:"payment_type_id"`
}

// TableName sets the insert table name for this struct type
func (b *YLABranchPaymentType) TableName() string {
	return "branch_payment_types"
}
