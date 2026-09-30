package contactlessmodels

import "time"

// Site ... Site Database Model
type Site struct {
	ID         int       `gorm:"column:id;primary_key" json:"id"`
	LocationID string    `gorm:"column:location_id" json:"location_id"`
	SiteURL    string    `gorm:"column:siteurl" json:"siteurl"`
	IkentooID  string    `gorm:"column:ikentoo_id" json:"ikentoo_id"`
	UpdatedAt  time.Time `gorm:"column:updated_at" json:"updated_at"`
	CreatedAt  time.Time `gorm:"column:created_at" json:"created_at"`
}

// Instruction ... Instruction Database Model
type Instruction struct {
	ID          int       `gorm:"column:id;primary_key" json:"id"`
	LocationID  string    `gorm:"column:locationid" json:"locationid"`
	Instruction string    `gorm:"column:instruction" json:"instruction"`
	Status      int       `gorm:"column:status" json:"status"`
	UpdatedAt   time.Time `gorm:"column:updated_at" json:"updated_at"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"created_at"`
}

// TableName sets the insert table name for this struct type
func (s *Site) TableName() string {
	return "sites"
}

// TableName sets the insert table name for this struct type
func (i *Instruction) TableName() string {
	return "instructions"
}

// ContactlessToken ... Contactless Token
type ContactlessToken struct {
	Token          string `json:"token"`
	BID            string `json:"blid"`
	BIDOverride    string `json:"businessoverride"`
	MenuID         string `json:"menu"`
	MenuIDOverride string `json:"menuoverride"`
	BaseID         string `json:"baseid"`
}

// ContactlessCategory ... Woocommerce Category
type ContactlessCategory struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Parent      int    `json:"parent"`
	Description string `json:"description"`
}

// ContactlessSimpleItem ... Contactless Woocommerce Simple Item
type ContactlessSimpleItem struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// ContactlessAttribute ... Contactless Woocommerce Simple Item
type ContactlessAttribute struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	Type string `json:"type"`
}

// ContactlessAttributeTerm ... Contactless Woocommerce Attribute Term
type ContactlessAttributeTerm struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	Desc string `json:"description"`
}

// ContactLessDescriptionUpdate ... Post Object For Item Description
type ContactLessDescriptionUpdate struct {
	SKU         string `json:"sku"`
	Description string `json:"description"`
}
