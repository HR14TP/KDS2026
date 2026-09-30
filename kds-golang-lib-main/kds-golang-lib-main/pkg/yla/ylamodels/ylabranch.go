package ylamodels

import (
	"time"
)

type YLABranch struct {
	ID                    int                    `gorm:"column:id;primary_key" json:"id"`
	CompanyID             int64                  `gorm:"column:company_id" json:"company_id"`
	ContactNumber         string                 `gorm:"column:contact_number" json:"contact_number"`
	Email                 string                 `gorm:"column:email" json:"email"`
	AddressLine1          string                 `gorm:"column:address_line_1" json:"address_line_1"`
	Location              string                 `gorm:"column:location" json:"location"`
	SubDomainURL          string                 `gorm:"column:sub_domain_url" json:"sub_domain_url"`
	IkentooID             int64                  `gorm:"column:ikentoo_id" json:"ikentoo_id"`
	DisplayName           string                 `gorm:"column:display_name" json:"display_name"`
	ContactName           string                 `gorm:"column:contact_name" json:"contact_name"`
	Active                int                    `gorm:"column:active" json:"active"`
	DeletedAt             time.Time              `gorm:"column:deleted_at" json:"-"`
	CreatedAt             time.Time              `gorm:"column:created_at" json:"-"`
	UpdatedAt             time.Time              `gorm:"column:updated_at" json:"-"`
	Message               string                 `gorm:"column:message" json:"message"`
	PfMerchantID          string                 `gorm:"column:pf_merchant_id" json:"pf_merchant_id"`
	PfMerchantKey         string                 `gorm:"column:pf_merchant_key" json:"pf_merchant_key"`
	PfConfirmationAddress string                 `gorm:"column:pf_confirmation_address" json:"pf_confirmation_address"`
	PfPassphrase          string                 `gorm:"column:pf_passphrase" json:"pf_passphrase"`
	Name                  string                 `gorm:"column:name" json:"name"`
	City                  string                 `gorm:"column:city" json:"city"`
	IkentooBusinessID     string                 `gorm:"column:ikentoo_business_id" json:"ikentoo_business_id"`
	AddressLine2          string                 `gorm:"column:address_line_2" json:"address_line_2"`
	Lat                   string                 `gorm:"column:lat" json:"lat"`
	Lng                   string                 `gorm:"column:lng" json:"lng"`
	StatisticGroups       string                 `gorm:"column:statistic_groups" json:"statistic_groups"`
	Deliveries            int                    `gorm:"column:deliveries" json:"deliveries"`
	MenuID                int                    `gorm:"column:menu_id" json:"menu_id"`
	Distance              float32                `gorm:"column:distance" json:"distance"`
	Collection            int                    `gorm:"column:collection" json:"collection"`
	Showextra             int                    `gorm:"column:showextra" json:"showextra"`
	IkentooVersion        int                    `gorm:"column:ikentoo_version" json:"ikentoo_version"`
	DeliveryItem          YLAItem                `gorm:"-" json:"deliveryitem"`
	TipItem               YLAItem                `gorm:"-" json:"tipitem"`
	DeliveryMinimum       int                    `gorm:"column:delivery_minimum" json:"delivery_minimum"`
	EnableRating          int                    `gorm:"column:enable_rating" json:"enable_rating"`
	DeliverySKU           string                 `gorm:"column:delivery_sku" json:"delivery_sku"`
	TipSKU                string                 `gorm:"column:tip_sku" json:"tip_sku"`
	EnableOnTheGo         int                    `gorm:"column:enable_onthego" json:"enable_onthego"`
	Menu                  YLAMenu                `gorm:"-"`
	Hours                 []YLABusinessHour      `gorm:"foreignkey:BranchID" json:"Hours"`
	BranchPaymentType     []YLABranchPaymentType `gorm:"-" json:"branch_payment_type"`
	ZapperMerchantID      string                 `gorm:"column:zapper_merchantid" json:"zapper_merchantid"`
	ZapperSiteID          string                 `gorm:"column:zapper_siteid" json:"zapper_siteid"`
	VoucherItemID         string                 `gorm:"column:voucher_itemid" json:"voucher_itemid"`
	VoucherRation         float32                `gorm:"column:voucher_ratio" json:"voucher_ratio"`
}

// TableName sets the insert table name for this struct type
func (b *YLABranch) TableName() string {
	return "branches"
}
