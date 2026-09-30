package kdsmodels

type KDSItems struct {
	ID                   int           `gorm:"column:id;primary_key" json:"id"`
	Qty                  int           `gorm:"column:qty" json:"qty"`
	Name                 string        `gorm:"column:name" json:"name"`
	Status               string        `gorm:"column:status" json:"status"`
	KDSScreenID          int           `gorm:"column:kds_screen_id" json:"kds_screen_id"`
	KDSOrderID           int           `gorm:"column:kds_order_id" json:"kds_order_id"`
	Created              string        `gorm:"column:created" json:"created"`
	Screen               KDSScreens    `gorm:"-" json:"screen"`
	SubItems             []KDSSubitems `gorm:"-" json:"subitems"`
	IsKiddies            int           `gorm:"column:iskiddies" json:"iskiddies"`
	PrepMins             int           `gorm:"column:prep_mins" json:"prep_mins"`
	PrepSecs             int           `gorm:"column:prep_secs" json:"prep_secs"`
	SubOrder             int           `gorm:"column:suborder" json:"suborder"`
	Reference            string        `gorm:"-" json:"reference"`
	// Occurrence disambiguates identical items within the same order (same
	// Name+Qty, e.g. two "Rump Steak 300gr" on one ticket) when syncing a
	// status change to another terminal. Every terminal has its own
	// auto-increment ids, so a remote change can only be addressed by
	// Name+Qty - which is ambiguous whenever an order has more than one
	// matching item. Occurrence is each item's 1-based rank (by id, i.e.
	// creation order) among same-Name+Qty siblings in the same order,
	// computed identically on both terminals since they process items from
	// the same feed in the same order. Not persisted - computed fresh at
	// send and receive time.
	Occurrence           int           `gorm:"-" json:"occurrence"`
	QtyRaw               float64       `gorm:"column:qty_raw" json:"qty_raw"`
	InProduction         bool          `gorm:"column:in_production" json:"in_production"`
	ProductionStartedAt  *string       `gorm:"column:production_started_at" json:"production_started_at"`
	ProductionEndedAt    *string       `gorm:"column:production_ended_at" json:"production_ended_at"`
	// PushedToOds marks that this item has been deliberately announced as
	// ready on an Order Display Screen terminal - distinct from Status ==
	// "COMPLETE", which only reflects the kitchen's own internal tick.
	PushedToOds          bool          `gorm:"column:pushed_to_ods" json:"pushed_to_ods"`
	PushedToOdsAt        *string       `gorm:"column:pushed_to_ods_at" json:"pushed_to_ods_at"`
	// SentToTable marks that a completed item has physically been carried
	// out to the table - distinct from Status == "COMPLETE", which only
	// reflects the kitchen's own internal tick. Only ever set once an item
	// (and, on the frontend, only once its whole order) is already
	// COMPLETE.
	SentToTable          bool          `gorm:"column:sent_to_table" json:"sent_to_table"`
	SentToTableAt        *string       `gorm:"column:sent_to_table_at" json:"sent_to_table_at"`
}

func (p *KDSItems) TableName() string {
	return "kds_items"
}

type KDSExportItemsMySQL struct {
	ID          int                `gorm:"column:id;primary_key" json:"id"`
	LocalID     int                `gorm:"column:local_id;primary_key" json:"local_id"`
	TerminalID  int                `gorm:"column:terminal_id" json:"terminal_id"`
	Qty         int                `gorm:"column:qty" json:"qty"`
	Name        string             `gorm:"column:name" json:"name"`
	Status      string             `gorm:"column:status" json:"status"`
	KDSScreenID int                `gorm:"column:kds_screen_id" json:"kds_screen_id"`
	KDSOrderID  int                `gorm:"column:kds_order_id" json:"kds_order_id"`
	IsKiddies   int                `gorm:"column:iskiddies" json:"iskiddies"`
	PrepMins    int                `gorm:"column:prep_mins" json:"prep_mins"`
	PrepSecs    int                `gorm:"column:prep_secs" json:"prep_secs"`
	SubOrder    int                `gorm:"column:suborder" json:"suborder"`
	Created     string             `gorm:"column:created" json:"created"`
	CreatedAt   string             `gorm:"column:created_at" json:"created_at"`
	Screen      KDSScreens         `gorm:"-" json:"screen"`
	SubItems    []KDSSubitemsMySQL `gorm:"-" json:"subitems"`
}

func (p *KDSExportItemsMySQL) TableName() string {
	return "kds_items"
}
