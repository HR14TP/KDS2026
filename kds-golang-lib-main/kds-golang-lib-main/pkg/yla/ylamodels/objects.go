package ylamodels

type YLAClaimVoucherInput struct {
	SKU        string `json:"SKU"`
	LOCATIONID string `json:"LOCATIONID"`
	IKENTOOID  string `json:"IKENTOOID"`
	EMAIL      string `json:"EMAIL"`
}

type YLAUserResult struct {
	Data YLAUser `json:"data"`
}

type YLAUserOffers struct {
	Redemptions []YLAGiftRedemption `json:"redemptions"`
	Offers      []YLAOffer          `json:"offers"`
	Stats       YLAPurchaseStats    `json:"stats"`
}

type YLAUserOffersExtension struct {
	Redemptions []YLAGiftRedemption `json:"redemptions"`
	Offers      []YLAOffer          `json:"offers"`
	Stats       YLAPurchaseStats    `json:"stats"`
	User        YLAUser             `json:"user"`
	Branch      YLABranch           `json:"branch"`
}

type YLAPurchaseStats struct {
	PurchaseByAccGroup interface{}         `json:"purchaseByAccGroup"`
	PurchaseCount      float64             `json:"purchaseCount"`
	LastPurchase       string              `json:"lastPurchase"`
	PurchaseAmount     float64             `json:"purchaseAmount"`
	TotalSpend         float64             `json:"totalspend"`
	Withdrawals        interface{}         `json:"withdrawals"`
	Deductions         []int               `json:"deductions"`
	Claims             []YLAGiftRedemption `json:"claims"`
}

type YLAOrderCallback struct {
	Message            string `json:"message"`
	Origin             string `json:"origin"`
	OrderID            string `json:"order_id"`
	AccountID          string `json:"account_id"`
	BusinessLocationID string `json:"business_location_id"`
	AccountIdentifier  string `json:"account_identifier"`
	Status             string `json:"status"`
}

type IkentooConsumer struct {
	FirstName         string `json:"first_name"`
	LastName          string `json:"last_name"`
	Email             string `json:"email"`
	IkentooBusinessID int    `json:"business_id"`
	PhoneNumber1      string `json:"phone_number1"`
	CompanyName       string `json:"company_name"`
}

type IkentooPunchCardInput struct {
	UserID            int    `json:"user_id"`
	UUID              string `json:"uuid"`
	IkentooBusinessID int    `json:"ikentoo_business_id"`
	Email             string `json:"email"`
}

type IkentooItemAPIResult struct {
	Results []IkentooItemAPI `json:"results"`
	Amount  int              `json:"amount"`
	Offset  int              `json:"offset"`
	Total   int              `json:"total"`
}

type IkentooItemAPI struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	SKU      string `json:"sku"`
	Active   bool   `json:"active"`
	Disabled bool   `json:"disabled"`
}

type IkentooUser struct {
	ID                string `json:"id"`
	Email             string `json:"email"`
	Password          string `json:"password"`
	Cell              string `json:"cell"`
	Name              string `json:"first_name"`
	Surname           string `json:"last_name"`
	PostalAddress     string `json:"address1"`
	IkentooID         string `json:"ikentoo_id"`
	Gender            int    `json:"gender"`
	IkentooBusinessID int    `json:"ikentoo_business_id"`
	AddressLine1      string `json:"address2"`
	City              string `json:"city"`
	Zip               string `json:"zip"`
	OsUserID          string `json:"os_user_id"`
	OsPushToken       string `json:"os_push_token"`
}

type LookupAddress struct {
	DisplayName string  `json:"display_name"`
	Latitude    string  `json:"lat"`
	Longitude   string  `json:"lon"`
	Importance  float32 `json:"importance"`
}

type LookupAddressDetails struct {
	Latitude  float32 `json:"latitude"`
	Longitude float32 `json:"longitude"`
}

type YLAFeedbackObject struct {
	Feedback  string `json:"feedback"`
	Stars     int    `json:"stars"`
	IkentooID string `json:"ikentoo_id"`
	UserID    int    `json:"user_id"`
}

type YLAOrderInstruction struct {
	OrderID      string         `json:"order_id"`
	DeliveryType string         `json:"delivery_type"`
	Address      YLAUserAddress `json:"address"`
	HasPaid      bool           `json:"haspaid"`
	Comment      string         `json:"comment"`
}

type YLAOneSingalUser struct {
	UserID      int    `json:"user_id"`
	OsUserID    string `json:"os_user_id"`
	OsPushToken string `json:"os_push_token"`
}

type IkentooOrder struct {
	BusinessLocationID int                `json:"business_location_id"`
	ExternalReference  string             `json:"external_reference"`
	DeliveryMode       string             `json:"delivery_mode"`
	PaymentAmount      float64            `json:"payment_amount"`
	Customer           OrderCustomer      `json:"customer"`
	Items              []OrderItemIkentoo `json:"items"`
	Profile            string             `json:"profile"`
	Comment            string             `json:"comment"`
}

type OrderCustomer struct {
	ExternalReference string `json:"external_reference"`
	FirstName         string `json:"first_name"`
	LastName          string `json:"last_name"`
	Address1          string `json:"address1"`
	Address2          string `json:"address2"`
	Zip               string `json:"zip"`
	City              string `json:"city"`
	Country           string `json:"country"`
	Salutation        string `json:"salutation"`
	CompanyName       string `json:"company_name"`
	Email             string `json:"email"`
}

type OrderItemIkentoo struct {
	ItemSku       string            `json:"item_sku"`
	ItemName      string            `json:"item_name"`
	Quantity      int               `json:"quantity"`
	ItemPrice     float64           `json:"item_price"`
	SubItems      []SubItems        `json:"sub_items"`
	ItemModifiers []ProductModifier `json:"item_modifiers"`
}

type SubItems struct {
	ItemSku   string  `json:"item_sku"`
	ItemPrice float64 `json:"item_price"`
	ItemName  string  `json:"item_name"`
}
type SubItemsInstruction struct {
	ItemSku   string  `json:"item_sku"`
	ItemPrice float64 `json:"item_price"`
	ItemName  string  `json:"item_name"`
}

type ProductModifier struct {
	ModID string `json:"mod_id"`
}

type YLAOrderRepeat struct {
	Order      YLAOrder            `json:"order"`
	Items      []YLAItem           `json:"items"`
	OrderSeqs  []YLAOrderSequences `json:"ordersequences"`
	OrderItems []YLAOrderItem      `json:"orderitems"`
}

type YLAIkentooOrder struct {
	TipAmount               string  `json:"tip_amount"`
	ScheduledCollectionTime string  `json:"scheduled_collection_time"`
	Status                  string  `json:"status"`
	DeliveryMode            string  `json:"delivery_mode"`
	Comment                 string  `json:"comment"`
	PaymentAmount           float64 `json:"payment_amount"`
	ID                      string  `json:"id"`
	Origin                  string  `json:"origin"`
	AccountID               string  `json:"account_id"`
	CollectionCode          string  `json:"collection_code"`
	TableNumber             string  `json:"table_number"`
	PaymentMethod           string  `json:"payment_method"`
}

type YLAV3INJECTION struct {
	Status string `json:"status"`
}

type YLAV3OrderCallback struct {
	OrderID           string `json:"thirdPartyReference"`
	Reason            string `json:"reason"`
	AccountID         int64  `json:"ikentooAccountId"`
	AccountIdentifier string `json:"ikentooAccountIdentifier"`
	Status            string `json:"status"`
}

type YLAAdHoc struct {
	MerchantID int    `json:"merchant_id"`
	Token      string `json:"token"`
	Timestamp  string `json:"timestamp"`
	Signature  string `json:"signature"`
	Body       string `json:"body"`
	URL        string `json:"url"`
}

type YLAUserLoginInput struct {
	Email             string `json:"email"`
	Phone             string `json:"phone"`
	IkentooBusinessID int    `json:"business_id"`
}

type IkentooGenericResponse struct {
	Msg string `json:"msg"`
}

type IkentooQRTokenResponse struct {
	QRToken string  `json:"QRToken"`
	User    YLAUser `json:"user"`
}

type IkentooConsumerCreated struct {
	ID                string `json:"id"`
	Vendor            string `json:"vendor"`
	Sex               string `json:"sex"`
	DOB               string `json:"dob"`
	FirstName         string `json:"first_name"`
	LastName          string `json:"last_name"`
	ExternalReference string `json:"external_reference"`
	Address1          string `json:"address1"`
	Address2          string `json:"address2"`
	Zip               string `json:"zip"`
	City              string `json:"city"`
	State             string `json:"state"`
	Country           string `json:"country"`
	Salutation        string `json:"salutation"`
	CompanyName       string `json:"company_name"`
	Email             string `json:"email"`
	PhoneNumber1      string `json:"phone_number1"`
	PhoneNumber2      string `json:"phone_number2"`
	TaxIdentifier     string `json:"tax_identifier"`
}

type RealTimeCallBack struct {
	NotificationType   string                  `json:"notificationType"`
	BusinessLocationId int                     `json:"businessLocationId"`
	Account            RealTimeCallBackAccount `json:"account"`
}
type RealTimeCallBackAccount struct {
	AccountNumber       string                              `json:"accountNumber"`
	AccountProfileCode  string                              `json:"accountProfileCode"`
	Type                string                              `json:"type"`
	AccountProfileName  string                              `json:"accountProfileName"`
	ExternalReferences  []RealTimeCallBackExternalReference `json:"externalReferences"`
	DeliveryInformation RealTimeCallBackDeliveryInformation `json:"deliveryInformation"`
}
type RealTimeCallBackExternalReference struct {
	Prefix    string `json:"prefix"`
	Reference string `json:"reference"`
}
type RealTimeCallBackDeliveryInformation struct {
	Enabled        bool   `json:"enabled"`
	DeliveryMode   string `json:"deliveryMode"`
	DeliveryState  string `json:"deliveryState"`
	CollectionCode string `json:"collectionCode"`
	DeliveryOrigin string `json:"deliveryOrigin"`
}
type POSVoucherInput struct {
	Price             int    `json:"price"`
	Title             string `json:"title"`
	Description       string `json:"description"`
	IkentooBusinessID string `json:"ikentoo_business_id"`
	IkentooId         string `json:"ikentoo_id"`
	Radius            int    `json:"radius"`
}
