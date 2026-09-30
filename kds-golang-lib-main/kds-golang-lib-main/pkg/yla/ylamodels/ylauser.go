package ylamodels

import "time"

type YLAUser struct {
	ID                    int              `gorm:"column:id;primary_key" json:"id"`
	Email                 string           `gorm:"column:email" json:"email"`
	Password              string           `gorm:"column:password" json:"password"`
	Cell                  string           `gorm:"column:cell" json:"cell"`
	Name                  string           `gorm:"column:name" json:"name"`
	Surname               string           `gorm:"column:surname" json:"surname"`
	PostalAddress         string           `gorm:"column:postal_address" json:"postal_address"`
	LastLoggedIn          time.Time        `gorm:"column:last_logged_in" json:"last_logged_in"`
	RegistrationTimestamp time.Time        `gorm:"column:registration_timestamp" json:"registration_timestamp"`
	UserTypeID            int              `gorm:"column:user_type_id" json:"user_type_id"`
	RememberToken         string           `gorm:"column:remember_token" json:"remember_token"`
	Archived              int              `gorm:"column:archived" json:"archived"`
	GUID                  string           `gorm:"column:guid" json:"guid"`
	OsUserID              string           `gorm:"column:os_user_id" json:"os_user_id"`
	OsPushToken           string           `gorm:"column:os_push_token" json:"os_push_token"`
	Pin                   string           `gorm:"column:pin" json:"pin"`
	IkentooID             string           `gorm:"column:ikentoo_id" json:"ikentoo_id"`
	Gender                int              `gorm:"column:gender" json:"gender"`
	Dob                   time.Time        `gorm:"column:dob" json:"dob"`
	IkentooBusinessID     string           `gorm:"column:ikentoo_business_id" json:"ikentoo_business_id"`
	PfToken               string           `gorm:"column:pf_token" json:"pf_token"`
	IkentooConsumerSpend  float64          `gorm:"column:ikentoo_consumer_spend" json:"ikentoo_consumer_spend"`
	AddressLine1          string           `gorm:"column:address_line_1" json:"address_line_1"`
	City                  string           `gorm:"column:city" json:"city"`
	Zip                   string           `gorm:"column:zip" json:"zip"`
	MembershipID          int              `gorm:"column:membership_id" json:"membership_id"`
	UserAccount           YLAAccount       `gorm:"-" json:"user_account"`
	Addresses             []YLAUserAddress `gorm:"-" json:"addresses"`
	Cards                 []YLAUserCard    `gorm:"-" json:"cards"`
}

// TableName sets the insert table name for this struct type
func (u *YLAUser) TableName() string {
	return "users"
}
