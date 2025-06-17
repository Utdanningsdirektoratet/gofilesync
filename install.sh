#!/bin/sh
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
x86_64) ARCH="amd64" ;;
aarch64 | arm64) ARCH="arm64" ;;
esac

URL="https://github.com/ajxudir/gofilesync/releases/latest/download/gofilesync_latest_${OS}_${ARCH}.tar.gz"
echo "Download binary from $URL"

if command -v curl >/dev/null 2>&1; then
    curl -sL "$URL" | tar -xz gofilesync
elif command -v wget >/dev/null 2>&1; then
    wget -qO- "$URL" | tar -xz gofilesync
else
    echo "Need curl or wget"
    exit 1
fi
