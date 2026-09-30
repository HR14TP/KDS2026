package main

import (
	"encoding/json"
	"net/http"

	"github.com/getsentry/sentry-go"
)

// GetconfigHandler always re-reads the config file fresh from disk rather
// than marshaling the shared package-level config var directly - that var
// is only as current as whichever caller last happened to invoke
// LoadConfiguration (previously just this process's own boot; now also
// announceOwnScreenPassAssignment/announceAsPassTerminal's periodic re-read,
// see pass_terminal_assignment.go), so relying on it here would make
// GetconfigHandler's own freshness an accident of unrelated code elsewhere
// rather than something guaranteed. The frontend's KDS.kdsconfig is
// otherwise only ever set once, at this terminal's own page load (KDS.init)
// - saving a change in Client Config and clicking Close Order right after,
// with no restart in between, needs this endpoint to hand back the value
// that was actually just saved, not whatever this process loaded at boot.
func GetconfigHandler(w http.ResponseWriter, r *http.Request) {
	current := LoadConfiguration(configLocation)
	js, err := json.Marshal(current)
	if err != nil {
		sentry.CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}
