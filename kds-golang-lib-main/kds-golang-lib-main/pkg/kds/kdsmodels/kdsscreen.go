package kdsmodels

type KDSScreens struct {
	ID   int    `gorm:"column:id;primary_key" json:"id"`
	Name string `gorm:"column:name" json:"name"`
	Port string `gorm:"column:port" json:"port"`
	// AssignedPassTerminal names the pass terminal responsible for bumping
	// this screen's orders (matches that terminal's own Client.Screen) -
	// empty means this screen bumps its own orders itself. Kept in sync
	// across every terminal's own local copy of this table via
	// announceOwnScreenPassAssignment's peer/NATS/UDP broadcast in
	// kds-webserver-main, since each terminal only knows its own screen's
	// assignment authoritatively from its own kdsconfig.json.
	AssignedPassTerminal string `gorm:"column:assigned_pass_terminal" json:"assigned_pass_terminal"`
}

func (p *KDSScreens) TableName() string {
	return "kds_screens"
}
