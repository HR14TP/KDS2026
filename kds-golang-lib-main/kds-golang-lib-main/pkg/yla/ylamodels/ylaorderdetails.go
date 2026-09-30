package ylamodels

type YLAOrderDetails struct {
	ID              int    `gorm:"column:id;primary_key" json:"id"`
	OrderID         int    `gorm:"column:order_id" json:"order_id"`
	DeliveryType    string `gorm:"column:delivery_type" json:"delivery_type"`
	DeliveryAddress string `gorm:"column:delivery_address" json:"delivery_address"`
	OrderNotes      string `gorm:"column:order_notes" json:"order_notes"`
	OrderReference  string `gorm:"column:order_reference" json:"order_reference"`
}

func (m *YLAOrderDetails) TableName() string {
	return "order_details"
}
