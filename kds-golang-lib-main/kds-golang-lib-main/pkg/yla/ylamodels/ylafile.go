package ylamodels

import "time"

type YLAFile struct {
	ID        int       `gorm:"column:id;primary_key" json:"id"`
	Name      string    `gorm:"column:name" json:"name"`
	Extension string    `gorm:"column:extension" json:"-"`
	Path      *string   `gorm:"column:path" json:"path"`
	MimeType  string    `gorm:"column:mime_type" json:"-"`
	Width     int       `gorm:"column:width" json:"-"`
	Height    int       `gorm:"column:height" json:"-"`
	Size      int       `gorm:"column:size" json:"-"`
	CreatedAt time.Time `gorm:"column:created_at" json:"-"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"-"`
}

// TableName sets the insert table name for this struct type
func (f *YLAFile) TableName() string {
	return "files"
}
