#!/usr/bin/env bash
set -euo pipefail

# Builds kds-webserver and kds-poller (with the new "in production" feature)
# and packages the Electron KDS app as a .deb.
#
# Run this ON the Debian/Ubuntu machine you intend to test on - building on
# the target machine avoids cross-compilation/glibc headaches entirely, and
# CGO is disabled below so no C cross-compiler toolchain is needed either.
#
# Usage:
#   ./build-and-package.sh            # builds for the machine's own arch
#   GOARCH=arm64 ./build-and-package.sh   # e.g. for an arm64 device instead

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WEBSERVER_DIR="$ROOT_DIR/kds-webserver-main/kds-webserver-main"
POLLER_DIR="$ROOT_DIR/kds-poller-main/kds-poller-main"
APP_DIR="$ROOT_DIR/kds-app-main/kds-app-main"

GOARCH="${GOARCH:-amd64}"

echo "== Checking toolchain =="
if ! command -v go >/dev/null 2>&1; then
	echo "Go not found. Install it first, e.g.:"
	echo "  sudo apt-get update && sudo apt-get install -y golang-go"
	echo "Needs Go 1.21+. If apt's version is older than that, grab a newer"
	echo "tarball from https://go.dev/dl/ instead (no root needed to unpack it)."
	exit 1
fi
echo "Go: $(go version)"

if ! command -v node >/dev/null 2>&1 || ! command -v npm >/dev/null 2>&1; then
	echo "Node/npm not found. Install Node 18 or 20 LTS (e.g. via nodesource or nvm) and re-run."
	exit 1
fi
echo "Node: $(node --version)"

if ! command -v dpkg-deb >/dev/null 2>&1 || ! command -v fakeroot >/dev/null 2>&1; then
	echo "Installing dpkg-deb/fakeroot (needed to build the .deb) ..."
	sudo apt-get update && sudo apt-get install -y dpkg fakeroot
fi

echo
echo "== Building kds-webserver (linux/$GOARCH) =="
cd "$WEBSERVER_DIR"
go mod tidy
CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH" go build -o files/kdswebserver-linux .
echo "-> $WEBSERVER_DIR/files/kdswebserver-linux"

echo
echo "== Building kds-poller (linux/$GOARCH) =="
cd "$POLLER_DIR"
go mod tidy
CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH" go build -o files/kdspoller-linux pkg/*.go
echo "-> $POLLER_DIR/files/kdspoller-linux"

echo
echo "== Building the Electron app (.deb only, skipping the rpm maker) =="
cd "$APP_DIR"
npm install
# Not `npx electron-forge` - on at least one machine we hit this, npx resolved
# a broken/incompatible global electron-forge@5 instead of this project's
# pinned @electron-forge/cli devDependency. Invoking the local package's own
# entrypoint script directly sidesteps npx's resolution entirely.
node node_modules/@electron-forge/cli/dist/electron-forge.js make --targets=@electron-forge/maker-deb

DEB_FILE="$(find "$APP_DIR/out/make/deb" -name '*.deb' | head -n1)"

echo
echo "================================================================"
echo "Build complete."
echo "  .deb package:      $DEB_FILE"
echo "  custom webserver:  $WEBSERVER_DIR/files/kdswebserver-linux"
echo "  custom poller:     $POLLER_DIR/files/kdspoller-linux"
echo
echo "Next step - installs the app, seeds both Go services, and relaunches KDS:"
echo "  sudo ./seed-custom-binaries.sh \"$WEBSERVER_DIR/files/kdswebserver-linux\" \"$POLLER_DIR/files/kdspoller-linux\" \"$DEB_FILE\""
echo
echo "This also doubles as the update path for an existing KDS install -"
echo "run it again any time you rebuild, on a fresh machine or an existing one."
echo "================================================================"
