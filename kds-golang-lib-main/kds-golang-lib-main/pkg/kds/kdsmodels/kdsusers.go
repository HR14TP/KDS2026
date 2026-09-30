package kdsmodels

import "time"

type KDSUser struct {
	ID            int       `gorm:"column:id;primary_key" json:"id"`
	Email         string    `gorm:"column:email" json:"email"`
	Password      string    `gorm:"column:password" json:"password"`
	UserTypeID    int       `gorm:"column:user_type_id" json:"user_type_id"`
	UserCreatedID int       `gorm:"column:user_created_id" json:"user_created_id"`
	CreatedAt     time.Time `gorm:"<-:false;column:created_at" json:"created_at"`
}

func (p *KDSUser) TableName() string {
	return "users"
}
