package ylamodels

import (
	"time"

	"github.com/reverent911/kds-golang-lib/pkg/generic/database"
	"gorm.io/gorm"
)

type YLAOrder struct {
	ID                    int            `gorm:"column:id;primary_key" json:"id"`
	UserID                int            `gorm:"column:user_id" json:"user_id"`
	OrderStatus           int            `gorm:"column:order_status" json:"order_status"`
	PaymentTypeID         string         `gorm:"column:payment_type_id" json:"payment_type_id"`
	BranchID              string         `gorm:"column:branch_id" json:"branch_id"`
	CreatedAt             time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt             time.Time      `gorm:"column:updated_at" json:"updated_at"`
	IkentooCollectionCode string         `gorm:"column:ikentoo_collection_code" json:"ikentoo_collection_code"`
	ExternalReference     string         `gorm:"column:external_reference" json:"external_reference"`
	PfStatus              string         `gorm:"column:pf_status" json:"pf_status"`
	IkentooStatus         string         `gorm:"column:ikentoo_status" json:"ikentoo_status"`
	PushNotification      string         `gorm:"column:push_notification" json:"push_notification"`
	PfResponse            string         `gorm:"column:pf_response" json:"pf_response"`
	IkentooMessages       string         `gorm:"column:ikentoo_messages" json:"ikentoo_messages"`
	IkentooResponse       string         `gorm:"column:ikentoo_response" json:"ikentoo_response"`
	DeliveryType          string         `gorm:"column:delivery_type" json:"delivery_type"`
	OrderItems            []YLAOrderItem `gorm:"foreignkey:OrderID" json:"orderitems"`
	Branch                YLABranch      `gorm:"-" json:"branch"`
	User                  YLAUser        `gorm:"-" json:"user"`
	OrderTotal            float64        `gorm:"-" json:"ordertotal"`
}
type YLAOrderItem struct {
	ID                              int                                 `gorm:"column:id;primary_key" json:"id"`
	OrderID                         int                                 `gorm:"column:order_id" json:"order_id"`
	ItemID                          string                              `gorm:"column:item_id" json:"item_id"`
	ItemSku                         string                              `gorm:"column:item_sku" json:"item_sku"`
	Qty                             int                                 `gorm:"column:qty" json:"qty"`
	Amount                          float64                             `gorm:"column:amount" json:"amount"`
	Vat                             float64                             `gorm:"column:vat" json:"vat"`
	OfferID                         int                                 `gorm:"column:offer_id" json:"offer_id"`
	Instructions                    string                              `gorm:"column:instructions" json:"instructions"`
	CreatedAt                       time.Time                           `gorm:"column:created_at" json:"created_at"`
	UpdatedAt                       time.Time                           `gorm:"column:updated_at" json:"updated_at"`
	Item                            YLAItem                             `gorm:"-" json:"item"`
	OrderItemSequences              []YLAOrderItemSequence              `gorm:"-" json:"orderitemsequences"`
	OrderItemProductionInstructions []YLAOrderItemProductionInstruction `gorm:"-" json:"orderitemproductioninstructions"`
}
type YLAOrderItemSequence struct {
	ID              int                `gorm:"column:id;primary_key" json:"id"`
	OrderID         int                `gorm:"column:order_id" json:"order_id"`
	OrderItemID     string             `gorm:"column:order_item_id" json:"order_item_id"`
	ItemSku         string             `gorm:"column:item_sku" json:"item_sku"`
	Item            YLAItem            `gorm:"-" json:"item"`
	SequenceValueID int                `gorm:"column:sequence_value_id" json:"sequence_value_id"`
	CreatedAt       time.Time          `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time          `gorm:"column:updated_at" json:"updated_at"`
	SequenceValues  []YLASequenceValue `gorm:"-" json:"sequencevalues"`
}
type YLAOrderItemProductionInstruction struct {
	ID                           int                             `gorm:"column:id;primary_key" json:"id"`
	OrderID                      int                             `gorm:"column:order_id" json:"order_id"`
	ItemSku                      string                          `gorm:"column:item_sku" json:"item_sku"`
	ProductionInstructionValueID int                             `gorm:"column:production_instruction_value_id" json:"production_instruction_value_id"`
	CreatedAt                    time.Time                       `gorm:"column:created_at" json:"created_at"`
	UpdatedAt                    time.Time                       `gorm:"column:updated_at" json:"updated_at"`
	ProductionValues             []YLAProductionInstructionValue `gorm:"-" json:"productionvalues"`
}
type YLAOrderSequences struct {
	ID              int    `gorm:"column:id;primary_key" json:"id"`
	OrderID         int    `gorm:"column:order_id;primary_key" json:"order_id"`
	ItemSKU         string `gorm:"column:item_sku;primary_key" json:"item_sku"`
	SequenceValueID int    `gorm:"column:sequence_value_id;primary_key" json:"sequence_value_id"`
}
type OrderExtra struct {
	Items        []YLAOrderItem                      `json:"items"`
	Sequences    []YLAOrderItemSequence              `json:"sequences"`
	Instructions []YLAOrderItemProductionInstruction `json:"instructions"`
	OrderID      int                                 `json:"order_id"`
}

type YLAPastOrder struct {
	ID                    int       `gorm:"column:id;primary_key" json:"id"`
	UserID                int       `gorm:"column:user_id" json:"user_id"`
	PaymentTypeID         string    `gorm:"column:payment_type_id" json:"payment_type_id"`
	BranchID              string    `gorm:"column:branch_id" json:"branch_id"`
	CreatedAt             time.Time `gorm:"column:created_at" json:"created_at"`
	IkentooCollectionCode string    `gorm:"column:ikentoo_collection_code" json:"ikentoo_collection_code"`
	ExternalReference     string    `gorm:"column:external_reference" json:"external_reference"`
	PfStatus              string    `gorm:"column:pf_status" json:"pf_status"`
	IkentooStatus         string    `gorm:"column:ikentoo_status" json:"ikentoo_status"`
	PushNotification      string    `gorm:"column:push_notification" json:"push_notification"`
	PfResponse            string    `gorm:"column:pf_response" json:"pf_response"`
	OrderStatus           int       `gorm:"column:order_status" json:"order_status"`
	Items                 []YLAItem `gorm:"-" json:"items"`
}

// TableName sets the insert table name for this struct type
func (o *YLAOrderItemSequence) TableName() string {
	return "order_item_sequences"
}

// TableName sets the insert table name for this struct type
func (o *YLAOrder) TableName() string {
	return "orders"
}
func (o *YLAOrder) CreateOrder(db *gorm.DB) {
	terminate := false
	if db == nil {
		terminate = true
		db = database.YLADebuggedConnection()
	}
	db.Create(o)
	if terminate {
		database.CloseDatabase(db)
	}
}

func (e *OrderExtra) UpdateOrder() {
	db := database.YLADebuggedConnection()
	for i := range e.Items {
		db.Create(&e.Items[i])
	}
	for j := range e.Sequences {
		db.Create(&e.Sequences[j])
	}
	for k := range e.Instructions {
		db.Create(&e.Instructions[k])
	}
	db.Model(&YLAOrder{}).Where("id = ?", e.OrderID).Update("order_status", 2)
	database.CloseDatabase(db)
}

// TableName sets the insert table name for this struct type
func (o *YLAPastOrder) TableName() string {
	return "orders"
}

// TableName sets the insert table name for this struct type
func (o *YLAOrderItem) TableName() string {
	return "order_items"
}

// TableName sets the insert table name for this struct type
func (o *YLAOrderItemProductionInstruction) TableName() string {
	return "order_item_production_instructions"
}

// TableName sets the insert table name for this struct type
func (o *YLAOrderSequences) TableName() string {
	return "order_item_sequences"
}
