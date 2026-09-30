package main

import (
	"encoding/json"

	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
)

// announceOwnScreenPassAssignment tells every other terminal on-site which
// pass terminal (if any) this terminal's own screen has been assigned to -
// each terminal only knows its own screen's assignment authoritatively from
// its own kdsconfig.json (Client.Passterminal, set via this terminal's
// config UI), so that fact has to be actively broadcast rather than looked
// up.
//
// Re-reads the config file fresh on every call rather than trusting
// ENV.Config, which this process only ever populates once, at its own
// boot (see InitConfSQLite in main.go) - nothing in the Electron app
// actually restarts these backend services after a config save (verified:
// IPC_SaveConfig -> services.setConfig only writes the file and publishes
// a CONFIGCHANGED notification, nothing more), so without this, neither
// the periodic re-announce nor the "Rediscover Pass Terminals" button
// could ever pick up a Pass Terminal assignment made after this process's
// own boot - only a full app restart would.
func announceOwnScreenPassAssignment() {
	current := LoadConfiguration(configLocation)
	if current.Client.Screen == "" {
		return
	}
	upsertScreenPassAssignment(current.Client.Screen, current.Client.Passterminal)
	assignment := kdsmodels.KDSScreenPassAssignment{
		Screen:       current.Client.Screen,
		Passterminal: current.Client.Passterminal,
	}
	js, _ := json.Marshal(&assignment)
	sendIfNeeded("DUPSCREENPASSASSIGNMENT", js)
	sendToPeers("screenpassassignmentremote", js)
	broadcastUDP("screenpassassignmentremote", js)
}

// announceAsPassTerminal tells every other terminal on-site that this
// terminal is available to be picked in a section's "Pass Terminal" config
// dropdown (getPassTerminalsHandler on the receiving end) - a no-op unless
// this terminal is actually in pass mode. Re-reads the config file fresh
// each call - see announceOwnScreenPassAssignment for why.
func announceAsPassTerminal() {
	current := LoadConfiguration(configLocation)
	if !current.Flags.Passmode || current.Client.Screen == "" {
		return
	}
	upsertPassTerminalAnnounce(current.Client.Screen)
	announce := kdsmodels.KDSPassTerminalAnnounce{Name: current.Client.Screen}
	js, _ := json.Marshal(&announce)
	sendIfNeeded("DUPPASSTERMINALANNOUNCE", js)
	sendToPeers("passterminalannounceremote", js)
	broadcastUDP("passterminalannounceremote", js)
}

// asyncAnnouncePassTerminalInfo re-announces both of the above periodically,
// on top of the boot-time call in main.go - covers a section terminal that
// boots before its assigned pass terminal has announced itself yet (or vice
// versa), and terminals that come back online after a UDP broadcast was
// missed.
func asyncAnnouncePassTerminalInfo() {
	announceOwnScreenPassAssignment()
	announceAsPassTerminal()
}

// requestPassTerminalRediscovery backs the Client Config screen's
// "Rediscover Pass Terminals" button (see rediscoverPassTerminalsHandler in
// close_order.go) - instead of a section terminal waiting up to 2 minutes
// for its own next asyncAnnouncePassTerminal tick, and up to another 2
// minutes for every OTHER terminal on-site to do the same, pressing the
// button announces this terminal right now and asks every other terminal to
// do the same right now too, over the same three delivery paths
// announceAsPassTerminal itself already uses (NATS screen1/screen2 mirrors,
// HTTP peers, UDP LAN broadcast) so it reaches terminals regardless of which
// path they're reachable over.
func requestPassTerminalRediscovery() {
	asyncAnnouncePassTerminalInfo()
	sendIfNeeded("DUPDISCOVERPING", []byte("{}"))
	sendToPeers("discoverpingremote", []byte("{}"))
	broadcastUDP("discoverpingremote", []byte("{}"))
}
