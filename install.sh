#!/bin/sh
# xmd installer: downloads the latest release binary for this OS/arch.
# Usage: sh -c "$(curl -fsSL https://raw.githubusercontent.com/benborla/xMarkdown/main/install.sh)"
set -e

REPO="benborla/xMarkdown"
BIN="xmd"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
  x86_64) arch=amd64 ;;
  aarch64 | arm64) arch=arm64 ;;
  *)
    echo "unsupported architecture: $arch" >&2
    exit 1
    ;;
esac
case "$os" in
  darwin | linux) ;;
  *)
    echo "unsupported OS: $os (darwin/linux only)" >&2
    exit 1
    ;;
esac

url="https://github.com/$REPO/releases/latest/download/${BIN}-${os}-${arch}"
tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT

echo "Downloading $url"
curl -fsSL "$url" -o "$tmp"
chmod +x "$tmp"

dest="/usr/local/bin"
if [ -w "$dest" ]; then
  mv "$tmp" "$dest/$BIN"
elif command -v sudo >/dev/null 2>&1; then
  echo "Installing to $dest (needs sudo)"
  sudo mv "$tmp" "$dest/$BIN"
else
  dest="$HOME/.local/bin"
  mkdir -p "$dest"
  mv "$tmp" "$dest/$BIN"
  case ":$PATH:" in
    *":$dest:"*) ;;
    *) echo "NOTE: add $dest to your PATH" ;;
  esac
fi

echo "Installed $dest/$BIN"
