#!/bin/sh
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
URL="https://github.com/ajxudir/gofilesync/releases/latest/download/gofilesync_${OS}_${ARCH}.tar.gz"
curl -L $URL | tar xz -C /usr/local/bin
