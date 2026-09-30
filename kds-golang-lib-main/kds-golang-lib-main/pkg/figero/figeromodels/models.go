package figeromodels

import "time"

type FIGEROUser struct {
	ID                    int       `gorm:"column:id;primary_key" json:"id"`
	Email                 string    `gorm:"column:email" json:"email"`
	Password              string    `gorm:"column:password" json:"password"`
	Cell                  string    `gorm:"column:cell" json:"cell"`
	Name                  string    `gorm:"column:name" json:"name"`
	Surname               string    `gorm:"column:surname" json:"surname"`
	PostalAddress         string    `gorm:"column:postal_address" json:"postal_address"`
	LastLoggedIn          time.Time `gorm:"column:last_logged_in" json:"last_logged_in"`
	RegistrationTimestamp time.Time `gorm:"column:registration_timestamp" json:"registration_timestamp"`
	UserTypeID            int       `gorm:"column:user_type_id" json:"user_type_id"`
	RememberToken         string    `gorm:"column:remember_token" json:"remember_token"`
	Archived              bool      `gorm:"column:archived" json:"archived"`
	GUID                  string    `gorm:"column:guid" json:"guid"`
	ParentUserID          int       `gorm:"column:parent_user_id" json:"parent_user_id"`
	ConsumerKey           string    `gorm:"column:consumer_key" json:"consumer_key"`
	ConsumerSecret        string    `gorm:"column:consumer_secret" json:"consumer_secret"`
	XeroPassword          string    `gorm:"column:xero_password" json:"xero_password"`
	TrialStartDate        time.Time `gorm:"column:trial_start_date" json:"trial_start_date"`
	PayfastToken          string    `gorm:"column:payfast_token" json:"payfast_token"`
	MobileApp             bool      `gorm:"column:mobile_app" json:"mobile_app"`
	UpdatedAt             time.Time `gorm:"column:updated_at" json:"updated_at"`
	XeroAuth              string    `gorm:"column:xero_auth" json:"xero_auth"`
}

func (p *FIGEROUser) TableName() string {
	return "users"
}

type FIGEROLocation struct {
	ID                     int       `gorm:"column:id;primary_key" json:"id"`
	OrgName                string    `gorm:"column:org_name" json:"org_name"`
	XeroOrgID              string    `gorm:"column:xero_org_id" json:"xero_org_id"`
	UserID                 int       `gorm:"column:user_id" json:"user_id"`
	FinYearEndMonth        string    `gorm:"column:fin_year_end_month" json:"fin_year_end_month"`
	Currency               string    `gorm:"column:currency" json:"currency"`
	Token                  string    `gorm:"column:token" json:"token"`
	TokenSecret            string    `gorm:"column:token_secret" json:"token_secret"`
	ExpiresIn              int       `gorm:"column:expires_in" json:"expires_in"`
	XeroOrgMUID            string    `gorm:"column:xero_org_muid" json:"xero_org_muid"`
	SessionHandle          string    `gorm:"column:session_handle" json:"session_handle"`
	AuthorizationExpiresIn int       `gorm:"column:authorization_expires_in" json:"authorization_expires_in"`
	Order                  int       `gorm:"column:order" json:"order"`
	Errors                 int       `gorm:"column:errors" json:"errors"`
	Oauth                  string    `gorm:"column:oauth" json:"oauth"`
	CreatedDate            string    `gorm:"column:created_date" json:"created_date"`
	CreatedAt              time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt              time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (p *FIGEROLocation) TableName() string {
	return "locations"
}

type FIGEROLocationDataDaily struct {
	ID          int       `gorm:"column:id;primary_key" json:"id"`
	LocationID  int       `gorm:"column:location_id" json:"location_id"`
	Income      float64   `gorm:"column:income" json:"income"`
	Gross       float64   `gorm:"column:gross" json:"gross"`
	Net         float64   `gorm:"column:net" json:"net"`
	CreatedDate string    `gorm:"column:created_date" json:"created_date"`
	Date        time.Time `gorm:"column:date" json:"date"`
	UpdatedDate time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (p *FIGEROLocationDataDaily) TableName() string {
	return "location_datas"
}

type FIGEROLocationMonthly struct {
	ID          int       `gorm:"column:id;primary_key" json:"id"`
	LocationID  int       `gorm:"column:location_id" json:"location_id"`
	Income      float64   `gorm:"column:income" json:"income"`
	Gross       float64   `gorm:"column:gross" json:"gross"`
	Net         float64   `gorm:"column:net" json:"net"`
	CreatedDate string    `gorm:"column:created_at" json:"created_at"`
	Date        time.Time `gorm:"column:date" json:"date"`
	UpdatedDate time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (p *FIGEROLocationMonthly) TableName() string {
	return "location_data_months"
}

type FIGEROYearlyStatsFromDaily struct {
	TotalIncome string
	TotalGross  string
	TotalNet    string
}
