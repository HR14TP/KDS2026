#!/usr/bin/env bash
set -euo pipefail

# Installs/updates KDS on this machine using the files bundled alongside this
# script. Just run it - no need to cd into this folder first, and no need to
# remember the seed-custom-binaries.sh arguments.
#
# Usage:
#   sudo ./install.sh

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

exec "$ROOT_DIR/seed-custom-binaries.sh" \
	"$ROOT_DIR/kdswebserver-linux" \
	"$ROOT_DIR/kdspoller-linux" \
	"$ROOT_DIR/kds2024_1.0.0_amd64.deb"
