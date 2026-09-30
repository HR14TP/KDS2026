package kdsmodels

import "time"

type KDSJobsMySQL struct {
	ID         int    `gorm:"column:id;primary_key" json:"id"`
	UserID     int    `gorm:"column:user_id" json:"user_id"`
	ReportType string `gorm:"column:report_type" json:"report_type"`
	ReportArgs string `gorm:"column:report_args" json:"report_args"`
	Trigger    string `gorm:"column:trigger" json:"trigger"`
	Status     string `gorm:"column:status" json:"status"`
}

func (p *KDSJobsMySQL) TableName() string {
	return "kds_jobs"
}

type KDSUserTypesMySQL struct {
	ID      int       `gorm:"column:id;primary_key" json:"id"`
	Name    string    `gorm:"column:name" json:"name"`
	Created time.Time `gorm:"<-:false;column:created" json:"created"`
}

func (p *KDSUserTypesMySQL) TableName() string {
	return "user_types"
}

type KDSUserTypePermissionsMySQL struct {
	ID           int       `gorm:"column:id;primary_key" json:"id"`
	UserTypeID   int       `gorm:"column:user_type_id" json:"user_type_id"`
	PermissionID int       `gorm:"column:permission_id" json:"permission_id"`
	Created      time.Time `gorm:"<-:false;column:created" json:"created"`
}

func (p *KDSUserTypePermissionsMySQL) TableName() string {
	return "user_type_permissions"
}

type KDSPermissionsMySQL struct {
	ID      int       `gorm:"column:id;primary_key" json:"id"`
	Name    string    `gorm:"column:name" json:"name"`
	Created time.Time `gorm:"<-:false;column:created" json:"created"`
}

func (p *KDSPermissionsMySQL) TableName() string {
	return "permissions"
}

type KDSAccountConfigMySQL struct {
	ID        int    `gorm:"column:id;primary_key" json:"id"`
	AccountID int    `gorm:"column:account_id" json:"account_id"`
	Config    string `gorm:"column:config" json:"config"`
}

func (p *KDSAccountConfigMySQL) TableName() string {
	return "account_config"
}

type KDSTerminalConfigV2MySQL struct {
	ID         int       `gorm:"column:id;primary_key" json:"id"`
	TerminalID int       `gorm:"column:terminal_id" json:"terminal_id"`
	Config     string    `gorm:"column:config" json:"config"`
	Created    time.Time `gorm:"<-:false;column:created" json:"created"`
}

func (p *KDSTerminalConfigV2MySQL) TableName() string {
	return "terminal_config"
}
