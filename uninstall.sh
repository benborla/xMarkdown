#!/bin/sh
# xmd uninstaller: removes the binary and optionally the config directory.
# Usage: sh -c "$(curl -fsSL https://raw.githubusercontent.com/benborla/xMarkdown/main/uninstall.sh)"
set -e

BIN="xmd"
found=""

for dest in /usr/local/bin "$HOME/.local/bin"; do
  if [ -f "$dest/$BIN" ]; then
    found="$dest/$BIN"
    if [ -w "$dest" ]; then
      rm "$found"
    elif command -v sudo >/dev/null 2>&1; then
      echo "Removing $found (needs sudo)"
      sudo rm "$found"
    else
      echo "cannot remove $found: no write permission and no sudo" >&2
      exit 1
    fi
    echo "Removed $found"
  fi
done

if [ -z "$found" ]; then
  echo "xmd not found in /usr/local/bin or ~/.local/bin — nothing to do"
  exit 0
fi

cfg="${XDG_CONFIG_HOME:-$HOME/.config}/xmd"
if [ -d "$cfg" ]; then
  printf "Remove config directory %s? [y/N] " "$cfg"
  read -r answer </dev/tty 2>/dev/null || answer=n
  case "$answer" in
    y | Y) rm -rf "$cfg" && echo "Removed $cfg" ;;
    *) echo "Kept $cfg" ;;
  esac
fi
