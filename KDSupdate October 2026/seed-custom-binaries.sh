#!/usr/bin/env bash
set -euo pipefail

# One-shot updater: run this on top of an existing KDS install to bring it up
# to this build - both the packaged Electron app (kds.js, the config screens,
# wimpy.css - anything that ships inside app.asar) and the custom
# kdswebserver/kdspoller binaries. Safe to run repeatedly, whether KDS has
# never been launched, is currently running, or was last updated with an
# older custom build.
#
# The .deb argument is what makes this a full update rather than just a
# binary swap: our JS/CSS/HTML changes live inside app.asar, which only a
# package install can replace - reseeding the Go binaries alone does not
# touch it. Pass a .deb whenever the Electron side changed (which is most of
# the time); omit it if you genuinely only rebuilt the Go services.
#
# The Electron app only downloads kdswebserver/kdspoller from S3 if they
# aren't already present in its userData "services" folder (see
# src/js/backend/services.js: services.checkService just checks
# fs.existsSync). Seeding them there first means the app uses ours instead.
#
# That's necessary but not sufficient to make KDS "use" the new binaries
# though - two things bit us before, both fixed here:
#
#   1. Overwriting a file in place while its old version is still running
#      (KDS already launched at least once) doesn't affect that already-running
#      process, and can even fail outright ("Text file busy"). We now stop
#      every KDS-related process first, and always write via a temp file +
#      atomic rename rather than truncating the target in place.
#   2. If this script is run with sudo (needed here for the .deb install, or
#      as part of a larger provisioning script), $HOME resolves to /root, so
#      the binaries would silently land in /root/.config/kds2024/services
#      instead of the real user's - KDS keeps using its old binaries and the
#      script reports success either way. We resolve the invoking user's
#      real home directory even under sudo.
#
# Usage: ./seed-custom-binaries.sh <path-to-kdswebserver-linux> <path-to-kdspoller-linux> [path-to-kds2024.deb]
#
# Examples:
#   # Full update (Electron app + both Go services) - the normal case:
#   sudo ./seed-custom-binaries.sh \
#       kds-webserver-main/kds-webserver-main/files/kdswebserver-linux \
#       kds-poller-main/kds-poller-main/files/kdspoller-linux \
#       kds-app-main/kds-app-main/out/make/deb/x64/kds2024_1.0.0_amd64.deb
#
#   # Go services only, no Electron/UI changes to install:
#   ./seed-custom-binaries.sh \
#       kds-webserver-main/kds-webserver-main/files/kdswebserver-linux \
#       kds-poller-main/kds-poller-main/files/kdspoller-linux

if [ $# -lt 2 ] || [ $# -gt 3 ]; then
	echo "Usage: $0 <path-to-kdswebserver-linux> <path-to-kdspoller-linux> [path-to-kds2024.deb]"
	exit 1
fi

WEBSERVER_SRC="$1"
POLLER_SRC="$2"
DEB_SRC="${3:-}"

for src in "$WEBSERVER_SRC" "$POLLER_SRC"; do
	if [ ! -f "$src" ]; then
		echo "Binary not found: $src" >&2
		exit 1
	fi
done
if [ -n "$DEB_SRC" ] && [ ! -f "$DEB_SRC" ]; then
	echo ".deb not found: $DEB_SRC" >&2
	exit 1
fi
if [ -n "$DEB_SRC" ] && [ "$(id -u)" -ne 0 ]; then
	echo "Installing $DEB_SRC needs root - re-run with sudo:" >&2
	echo "  sudo $0 $*" >&2
	exit 1
fi

TARGET_USER="${SUDO_USER:-$USER}"
TARGET_HOME="$(getent passwd "$TARGET_USER" | cut -d: -f6)"
if [ -z "$TARGET_HOME" ] || [ ! -d "$TARGET_HOME" ]; then
	echo "Could not resolve a home directory for user '$TARGET_USER'." >&2
	exit 1
fi

CONFIG_DIR="$TARGET_HOME/.config/kds2024"
SERVICES_DIR="$CONFIG_DIR/services"

echo "Target user:  $TARGET_USER"
echo "Services dir: $SERVICES_DIR"

echo
echo "== Stopping any running KDS processes =="
# pgrep -f matches whole command lines. Matching on bare keywords like
# "kdswebserver"/"kds2024" is a trap: our OWN argv contains those same
# substrings (the source paths we were given, e.g. ".../kdswebserver-linux"
# or ".../kds2024_1.0.0_amd64.deb"), and under sudo the separate monitor
# process sudo spawns shows that same command line too - excluding just our
# own $$ doesn't cover it. That self-match silently killed the script before
# it ever reached the dpkg install step. Matching the actual installed
# runtime paths instead - which our build-source arguments never resemble -
# avoids the ambiguity entirely rather than trying to enumerate every PID to
# exclude.
find_kds_pids() {
	pgrep -f "/usr/lib/kds2024/kds2024|$SERVICES_DIR/kds(webserver|poller|nats|tcpserver)" || true
}

PIDS="$(find_kds_pids)"
if [ -n "$PIDS" ]; then
	echo "Stopping: $PIDS"
	# shellcheck disable=SC2086
	kill $PIDS 2>/dev/null || true
	for _ in $(seq 1 20); do
		PIDS="$(find_kds_pids)"
		[ -z "$PIDS" ] && break
		sleep 0.5
	done
	PIDS="$(find_kds_pids)"
	if [ -n "$PIDS" ]; then
		echo "Still running after 10s, sending SIGKILL: $PIDS"
		# shellcheck disable=SC2086
		kill -9 $PIDS 2>/dev/null || true
		sleep 1
	fi
else
	echo "Nothing running."
fi

if [ -n "$DEB_SRC" ]; then
	echo
	echo "== Installing $DEB_SRC =="
	dpkg -i "$DEB_SRC" || apt-get install -f -y
fi

mkdir -p "$SERVICES_DIR"
chown "$TARGET_USER" "$SERVICES_DIR" 2>/dev/null || true

seed_binary() {
	local src="$1" name="$2"
	local dest="$SERVICES_DIR/$name"
	local tmp
	tmp="$(mktemp "$SERVICES_DIR/.$name.XXXXXX")"
	cp "$src" "$tmp"
	chmod +x "$tmp"
	chown "$TARGET_USER" "$tmp" 2>/dev/null || true
	# Atomic rename instead of overwriting $dest in place: this can't hit
	# "Text file busy" even if something still has the old file open, and
	# any process started after this line is guaranteed to see the new file.
	mv -f "$tmp" "$dest"
	echo "Seeded $name <- $src ($(stat -c%s "$dest") bytes)"
}

echo
echo "== Seeding binaries =="
seed_binary "$WEBSERVER_SRC" "kdswebserver"
seed_binary "$POLLER_SRC" "kdspoller"

echo
if [ "$TARGET_USER" = "$USER" ]; then
	echo "== Relaunching KDS =="
	if command -v kds2024 >/dev/null 2>&1; then
		nohup kds2024 >/dev/null 2>&1 &
		disown
		echo "Launched kds2024."
	else
		echo "Could not find 'kds2024' on PATH - start it manually."
	fi
else
	echo "Updated for $TARGET_USER, but this script is running as $USER (root,"
	echo "via sudo), so it can't safely relaunch the GUI app for that user's"
	echo "session (display/auth environment would be missing). Log in as"
	echo "$TARGET_USER (or from its desktop session) and launch kds2024 manually."
fi
