package ylamodels

import "time"

type PaymentITN struct {
	ID            int       `gorm:"column:id;primary_key" json:"id"`
	MerchantID    string    `gorm:"column:merchant_id" json:"merchant_id"`
	MPaymentID    string    `gorm:"column:m_payment_id" json:"m_payment_id"`
	PaymentStatus string    `gorm:"column:payment_status" json:"payment_status"`
	PFPaymentID   string    `gorm:"column:pf_payment_id" json:"pf_payment_id"`
	Token         string    `gorm:"column:token" json:"token"`
	Amount        string    `gorm:"column:amount" json:"amount_gross"`
	IkentooID     string    `gorm:"column:ikentoo_id" json:"ikentoo_id"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
}

func (p *PaymentITN) TableName() string {
	return "payment_itn"
}

type ZapperPaymentITN struct {
	Status                   int       `json:"status"`
	TenderedAmount           int       `json:"tenderedAmount"`
	InvoiceExternalReference string    `json:"invoiceExternalReference"`
	PaymentReference         string    `json:"paymentReference"`
	Created                  time.Time `json:"paymentUTCDate"`
}
