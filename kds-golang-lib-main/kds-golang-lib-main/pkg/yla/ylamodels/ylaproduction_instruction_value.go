package ylamodels

import (
	"time"
)

type YLAProductionInstructionValue struct {
	ID                      int       `gorm:"column:id;primary_key" json:"id"`
	ProductionInstructionID int       `gorm:"column:production_instruction_id" json:"production_instruction_id"`
	Name                    string    `gorm:"column:name" json:"name"`
	DeletedAt               time.Time `gorm:"column:deleted_at" json:"deleted_at"`
	CreatedAt               time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt               time.Time `gorm:"column:updated_at" json:"updated_at"`
	IkentooBusinessID       *string   `gorm:"column:ikentoo_business_id" json:"ikentoo_business_id"`
}

// TableName sets the insert table name for this struct type
func (p *YLAProductionInstructionValue) TableName() string {
	return "production_instruction_values"
}
