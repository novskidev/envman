#!/bin/sh
# envman installer - downloads the release binary from GitHub.
# Usage: curl -fsSL https://raw.githubusercontent.com/novian/envman/main/install.sh | sh
set -e

REPO="${ENVMAN_REPO:-novian/envman}"
VERSION="${1:-latest}"
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$OS" in
  linux|darwin) ;;
  *) echo "error: unsupported OS: $OS" >&2; exit 1 ;;
esac

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "error: unsupported arch: $ARCH" >&2; exit 1 ;;
esac

if [ "$VERSION" = "latest" ]; then
  URL="https://github.com/$REPO/releases/latest/download/envman-${OS}-${ARCH}"
else
  URL="https://github.com/$REPO/releases/download/v${VERSION}/envman-${OS}-${ARCH}"
fi

DEST="${ENVMAN_INSTALL_DIR:-/usr/local/bin}"
if [ ! -w "$DEST" ]; then
  DEST="$HOME/.local/bin"
  mkdir -p "$DEST"
fi

echo "downloading $URL"
TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT
curl -fsSL "$URL" -o "$TMP"
install -m 0755 "$TMP" "$DEST/envman"
echo "installed envman to $DEST/envman"