package ylamodels

type YLAV3Order struct {
	BusinessLocationID                int               `json:"businessLocationId"`
	ThirdPartyReference               string            `json:"thirdPartyReference"`
	EndpointID                        string            `json:"endpointId"`
	CustomerInfo                      YLAV3CustomerInfo `json:"customerInfo"`
	OrderNote                         string            `json:"orderNote"`
	MaxTimeToAttemptOrderDeliverToPos int               `json:"maxTimeToAttemptOrderDeliverToPos"`
	//StaffId                           int               `json:"staffId"`
	AccountProfileCode             string       `json:"accountProfileCode"`
	ScheduledTimeForOrderAsIso8601 string       `json:"scheduledTimeForOrderAsIso8601"`
	Payment                        YLAV3Payment `json:"payment"`
	CollectionCode                 string       `json:"collectionCode"`
	OrderCollectionTimeAsIso8601   string       `json:"orderCollectionTimeAsIso8601"`
	Items                          []YLAV3Items `json:"items"`
	TakeAway                       bool         `json:"takeAway"`
	DeliveryAddress                YLAV3Address `json:"deliveryAddress"`
	ClientCount                    int          `json:"clientCount"`
}

type YLAV3CustomerInfo struct {
	FirstName           string `json:"firstName"`
	LastName            string `json:"lastName"`
	Email               string `json:"email"`
	ContactNumberAsE164 string `json:"contactNumberAsE164"`
	Notes               string `json:"notes"`
}

type YLAV3Payment struct {
	PaymentMethod string  `json:"paymentMethod"`
	PaymentAmount float64 `json:"paymentAmount"`
	TipAmount     float64 `json:"tipAmount"`
}

type YLAV3Items struct {
	Quantity        int              `json:"quantity"`
	Sku             string           `json:"sku"`
	CustomItemName  string           `json:"customItemName"`
	CustomItemPrice float64          `json:"customItemPrice"`
	Modifiers       []YLAV3Modifiers `json:"modifiers"`
	DiscountCode    string           `json:"discountCode"`
	Course          int              `json:"course"`
	SubItems        []YLAV3SubItems  `json:"subItems"`
}

type YLAV3Modifiers struct {
	ModifierId string `json:"modifierId"`
}

type YLAV3SubItems struct {
	Quantity        int              `json:"quantity"`
	Sku             string           `json:"sku"`
	CustomItemName  string           `json:"customItemName"`
	CustomItemPrice float64          `json:"customItemPrice"`
	Modifiers       []YLAV3Modifiers `json:"modifiers"`
}

type YLAV3Address struct {
	AddressLine1 string `json:"addressLine1"`
	AddressLine2 string `json:"addressLine2"`
	Zip          string `json:"zip"`
	City         string `json:"city"`
}
