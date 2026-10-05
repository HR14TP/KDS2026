package kdsmodels

// KDSKnownTerminals records every terminal this one has heard announce its
// own screen name - not just pass-mode ones (see KDSPassTerminals, which is
// scoped to that narrower pass-terminal-dropdown use case). Populated from
// the source IP of the existing screenpassassignmentremote broadcast every
// terminal already sends periodically (see announceOwnScreenPassAssignment)
// - no separate announce message needed, this just also records where it
// came from. Backs the "Additional KDS Terminal IPs" auto-populated list on
// the Client Config screen, so a site on static IPs never has to manually
// type/maintain that comma-separated list by hand.
type KDSKnownTerminals struct {
	ID       int    `gorm:"column:id;primary_key" json:"id"`
	Screen   string `gorm:"column:screen" json:"screen"`
	IP       string `gorm:"column:ip" json:"ip"`
	LastSeen string `gorm:"column:last_seen" json:"last_seen"`
}

func (p *KDSKnownTerminals) TableName() string {
	return "kds_known_terminals"
}
