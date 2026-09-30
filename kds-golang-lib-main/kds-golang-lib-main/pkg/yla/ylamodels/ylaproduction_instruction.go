package ylamodels

import (
	"time"
)

type YLAProductionInstruction struct {
	ID                         int                             `gorm:"column:id;primary_key" json:"id"`
	Name                       string                          `gorm:"column:name" json:"name"`
	Type                       string                          `gorm:"column:type" json:"type"`
	IkentooBusinessID          string                          `gorm:"column:ikentoo_business_id" json:"ikentoo_business_id"`
	DeletedAt                  time.Time                       `gorm:"column:deleted_at" json:"deleted_at"`
	CreatedAt                  time.Time                       `gorm:"column:created_at" json:"created_at"`
	UpdatedAt                  time.Time                       `gorm:"column:updated_at" json:"updated_at"`
	ProductionInstructionValue []YLAProductionInstructionValue `gorm:"foreignkey:ProductionInstructionID"`
}

// TableName sets the insert table name for this struct type
func (p *YLAProductionInstruction) TableName() string {
	return "production_instructions"
}
