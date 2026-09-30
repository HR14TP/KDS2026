package kdsmodels

import "time"

type KDSOrders struct {
	ID            int        `gorm:"column:id;primary_key" json:"id"`
	Table         string     `gorm:"column:table" json:"table"`
	Consumer      string     `gorm:"column:consumer" json:"consumer"`
	Delivery      string     `gorm:"column:delivery" json:"delivery"`
	Waiter        string     `gorm:"column:waiter" json:"waiter"`
	Status        string     `gorm:"column:status" json:"status"`
	PassStatus    string     `gorm:"column:pass_status" json:"pass_status"`
	Profile       string     `gorm:"column:profile" json:"profile"`
	Reference     string     `gorm:"column:reference" json:"reference"`
	OrderFortoday int        `gorm:"column:order_forday" json:"order_forday"`
	SubOrders     int        `gorm:"column:suborders" json:"suborders"`
	Created       string     `gorm:"column:created" json:"created"`
	Completed     *string    `gorm:"column:completed" json:"-"`
	// TransferNote is set when this order/tile was spun off by a line-item
	// transfer (see kds-poller's engine.go) - it names the table the item
	// came from, so the frontend can render a distinct banner above the
	// items without disturbing the normal Profile heading (e.g. "sit down").
	// Empty for every ordinary order.
	TransferNote  string     `gorm:"column:transfer_note" json:"transfer_note"`
	Items         []KDSItems `gorm:"-" json:"items"`
}

func (p *KDSOrders) TableName() string {
	return "kds_orders"
}

type KDSBackupOrders struct {
	ID             int                   `gorm:"column:id;primary_key" json:"id"`
	Table          string                `gorm:"column:table" json:"table"`
	Waiter         string                `gorm:"column:waiter" json:"waiter"`
	Status         string                `gorm:"column:status" json:"status"`
	Profile        string                `gorm:"column:profile" json:"profile"`
	OrderFortoday  int                   `gorm:"column:order_forday" json:"order_forday"`
	CreatedAtOnKDS time.Time             `gorm:"column:created_at_on_kds" json:"created_at_on_kds"`
	CreatedAt      time.Time             `gorm:"column:created_at" json:"created_at"`
	Created        time.Time             `gorm:"column:created" json:"created"`
	Reference      string                `gorm:"column:reference" json:"reference"`
	TerminalID     int                   `gorm:"column:terminal_id" json:"terminal_id"`
	Completed      time.Time             `gorm:"column:completed" json:"completed"`
	SubOrders      int                   `gorm:"column:suborders" json:"suborders"`
	PassStatus     string                `gorm:"column:pass_status" json:"pass_status"`
	Consumer       string                `gorm:"column:consumer" json:"consumer"`
	Delivery       string                `gorm:"column:delivery" json:"delivery"`
	LocalID        int                   `gorm:"column:local_id" json:"local_id"`
	Items          []KDSExportItemsMySQL `gorm:"-" json:"items"`
}

func (p *KDSBackupOrders) TableName() string {
	return "kds_orders"
}

type KDSExportOrders struct {
	ID            int                   `gorm:"column:id;primary_key" json:"id"`
	Table         string                `gorm:"column:table" json:"table"`
	Consumer      string                `gorm:"column:consumer" json:"consumer"`
	Delivery      string                `gorm:"column:delivery" json:"delivery"`
	Waiter        string                `gorm:"column:waiter" json:"waiter"`
	Status        string                `gorm:"column:status" json:"status"`
	PassStatus    string                `gorm:"column:pass_status" json:"pass_status"`
	Profile       string                `gorm:"column:profile" json:"profile"`
	Reference     string                `gorm:"column:reference" json:"reference"`
	OrderFortoday int                   `gorm:"column:order_forday" json:"order_forday"`
	SubOrders     int                   `gorm:"column:suborders" json:"suborders"`
	Created       time.Time             `gorm:"column:created" json:"created"`
	Completed     time.Time             `gorm:"column:completed" json:"completed"`
	Items         []KDSExportItemsMySQL `gorm:"-" json:"items"`
}

func (p *KDSExportOrders) TableName() string {
	return "kds_orders"
}
