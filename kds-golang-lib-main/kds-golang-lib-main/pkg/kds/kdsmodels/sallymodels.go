package kdsmodels

type SallyOrder struct {
	Id            string           `json:"id"`
	ProductCenter string           `json:"productCenter"`
	TableName     string           `json:"tableName"`
	WaiterName    string           `json:"waiterName"`
	DeviceID      string           `json:"deviceID"`
	DateTime      string           `json:"dateTime"`
	Notes         string           `json:"notes"`
	Profile       *string          `json:"profile"`
	ConsumerName  *string          `json:"consumerName"`
	Items         []SallyMainItems `json:"items"`
}

type SallyMainItems struct {
	Sku        string          `json:"sku"`
	Name       string          `json:"name"`
	Note       *string         `json:"note"`
	VoidReason *string         `json:"voidReason"`
	Qty        int             `json:"qty"`
	Price      int             `json:"price"`
	Seat       *int            `json:"seat"`
	Course     *int            `json:"course"`
	SubItems   []SallySubItems `json:"subItems"`
}

type SallySubItems struct {
	Qty   int    `json:"qty"`
	Name  string `json:"name"`
	Price int    `json:"price"`
}
