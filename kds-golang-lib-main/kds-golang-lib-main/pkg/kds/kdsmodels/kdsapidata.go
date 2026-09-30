package kdsmodels

import "time"

type KDSApiDataAccount struct {
	ID        int                  `gorm:"column:id;primary_key" json:"id"`
	UserID    int                  `gorm:"column:user_id" json:"user_id"`
	Name      string               `gorm:"column:name" json:"name"`
	CreatedAt time.Time            `gorm:"column:created_at" json:"created_at"`
	Terminals []KDSApiDataTerminal `gorm:"-" json:"terminals"`
}

func (p *KDSApiDataAccount) TableName() string {
	return "accounts"
}

type KDSApiDataTerminal struct {
	ID        int                `gorm:"column:id;primary_key" json:"id"`
	AccountID int                `gorm:"column:account_id" json:"account_id"`
	Name      string             `gorm:"column:name" json:"name"`
	CreatedAt time.Time          `gorm:"column:created_at" json:"created_at"`
	Orders    []KDSApiDataOrders `gorm:"-" json:"orders"`
}

func (p *KDSApiDataTerminal) TableName() string {
	return "terminals"
}

type KDSApiDataOrders struct {
	ID             int               `gorm:"column:id;primary_key" json:"id"`
	Table          string            `gorm:"column:table" json:"table"`
	Waiter         string            `gorm:"column:waiter" json:"waiter"`
	Status         string            `gorm:"column:status" json:"status"`
	Profile        string            `gorm:"column:profile" json:"profile"`
	OrderFortoday  int               `gorm:"column:order_forday" json:"order_forday"`
	CreatedAtOnKDS time.Time         `gorm:"column:created_at_on_kds" json:"created_at_on_kds"`
	CreatedAt      time.Time         `gorm:"column:created_at" json:"created_at"`
	Reference      string            `gorm:"column:reference" json:"reference"`
	TerminalID     int               `gorm:"column:terminal_id" json:"terminal_id"`
	Completed      time.Time         `gorm:"column:completed" json:"completed"`
	SubOrders      int               `gorm:"column:suborders" json:"suborders"`
	Consumer       string            `gorm:"column:consumer" json:"consumer"`
	Delivery       string            `gorm:"column:delivery" json:"delivery"`
	Items          []KDSApiDataItems `gorm:"-" json:"items"`
}

func (p *KDSApiDataOrders) TableName() string {
	return "kds_orders"
}

type KDSApiDataItems struct {
	ID         int                  `gorm:"column:id;primary_key" json:"id"`
	TerminalID int                  `gorm:"column:terminal_id" json:"terminal_id"`
	Qty        int                  `gorm:"column:qty" json:"qty"`
	Name       string               `gorm:"column:name" json:"name"`
	Status     string               `gorm:"column:status" json:"status"`
	KDSOrderID int                  `gorm:"column:kds_order_id" json:"kds_order_id"`
	IsKiddies  int                  `gorm:"column:iskiddies" json:"iskiddies"`
	PrepMins   int                  `gorm:"column:prep_mins" json:"prep_mins"`
	PrepSecs   int                  `gorm:"column:prep_secs" json:"prep_secs"`
	SubOrder   int                  `gorm:"column:suborder" json:"suborder"`
	Created    string               `gorm:"column:created" json:"created"`
	CreatedAt  string               `gorm:"column:created_at" json:"created_at"`
	SubItems   []KDSApiDataSubitems `gorm:"-" json:"subitems"`
}

func (p *KDSApiDataItems) TableName() string {
	return "kds_items"
}

type KDSApiDataSubitems struct {
	ID         int       `gorm:"column:id;primary_key" json:"id"`
	TerminalID int       `gorm:"column:terminal_id" json:"terminal_id"`
	Qty        int       `gorm:"column:qty" json:"qty"`
	Name       string    `gorm:"column:name" json:"name"`
	Status     string    `gorm:"column:status" json:"status"`
	KDSItemID  int       `gorm:"column:kds_item_id" json:"kds_item_id"`
	CreatedAt  time.Time `gorm:"-" json:"created_at"`
}

func (p *KDSApiDataSubitems) TableName() string {
	return "kds_subitems"
}
