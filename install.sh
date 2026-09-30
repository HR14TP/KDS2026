#!/usr/bin/env bash
set -euo pipefail

# Installs/updates KDS on this machine using the files bundled alongside this
# script. Just run it - no need to cd into this folder first, and no need to
# remember the seed-custom-binaries.sh arguments.
#
# Usage:
#   sudo ./install.sh

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# This same folder gets copied to every KDS terminal we deploy to, so fail
# loudly and early here rather than letting seed-custom-binaries.sh hit a
# confusing half-done state partway through on some of them.

if [ "$(id -u)" -ne 0 ]; then
	echo "This needs root (it installs a .deb) - re-run with sudo:" >&2
	echo "  sudo $0" >&2
	exit 1
fi

# Globbed rather than a fixed filename so this doesn't need editing every
# time the version bumps (the app was also renamed kds2024 -> kdstabletpos,
# which changes the .deb's name too).
DEB_FILE="$(find "$ROOT_DIR" -maxdepth 1 -name '*.deb' | head -n1)"
if [ -z "$DEB_FILE" ]; then
	echo "No .deb found in $ROOT_DIR" >&2
	exit 1
fi
for bin in kdswebserver-linux kdspoller-linux; do
	if [ ! -f "$ROOT_DIR/$bin" ]; then
		echo "$bin not found in $ROOT_DIR - this folder was copied incompletely." >&2
		exit 1
	fi
done

# Different KDS terminals aren't guaranteed to be the same architecture (an
# amd64 kitchen PC and an arm64 tablet could both be running this from the
# same distributed folder) - catch a mismatch here with a clear message
# instead of letting it surface later as a cryptic "cannot execute binary
# file: Exec format error" out of a background service.
DEB_ARCH="$(dpkg-deb -f "$DEB_FILE" Architecture 2>/dev/null || echo unknown)"
HOST_ARCH="$(dpkg --print-architecture 2>/dev/null || echo unknown)"
if [ "$DEB_ARCH" != "unknown" ] && [ "$HOST_ARCH" != "unknown" ] && [ "$DEB_ARCH" != "$HOST_ARCH" ]; then
	echo "Architecture mismatch: $(basename "$DEB_FILE") is built for '$DEB_ARCH', but this machine is '$HOST_ARCH'." >&2
	echo "This distribution folder can't be used on this machine - get one built for '$HOST_ARCH' instead" >&2
	echo "(e.g. GOARCH=$HOST_ARCH ./build-and-package.sh on the '$HOST_ARCH' machine itself)." >&2
	exit 1
fi

exec "$ROOT_DIR/seed-custom-binaries.sh" \
	"$ROOT_DIR/kdswebserver-linux" \
	"$ROOT_DIR/kdspoller-linux" \
	"$DEB_FILE"
