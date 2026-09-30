package kdsmodels

// KDSPassTerminals is this terminal's local record of every pass-mode
// terminal it has heard announce itself on-site (see announceAsPassTerminal
// in kds-webserver-main) - purely so a section terminal's own config page
// can list them in its "Pass Terminal" dropdown. Not a source of truth for
// anything else; a terminal missing from here just hasn't announced itself
// (yet, or ever) to this particular terminal.
type KDSPassTerminals struct {
	ID       int    `gorm:"column:id;primary_key" json:"id"`
	Name     string `gorm:"column:name" json:"name"`
	LastSeen string `gorm:"column:last_seen" json:"last_seen"`
}

func (p *KDSPassTerminals) TableName() string {
	return "kds_pass_terminals"
}
