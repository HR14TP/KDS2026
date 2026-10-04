#!/usr/bin/env bash
set -euo pipefail

# Installs/updates KDS on this machine using the files bundled alongside this
# script. Just run it - no need to cd into this folder first, and no need to
# remember the seed-custom-binaries.sh arguments.
#
# Usage:
#   sudo ./install.sh

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

DEB_FILE="$(find "$ROOT_DIR" -maxdepth 1 -name '*.deb' | head -n1)"
if [ -z "$DEB_FILE" ]; then
	echo "No .deb file found alongside install.sh in $ROOT_DIR" >&2
	exit 1
fi

exec "$ROOT_DIR/seed-custom-binaries.sh" \
	"$ROOT_DIR/kdswebserver-linux" \
	"$ROOT_DIR/kdspoller-linux" \
	"$DEB_FILE"
