#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PLIST_SOURCE="$ROOT_DIR/mac/StatusApp/LaunchAgents/com.felipeveas.dotctl.statusapp.plist"
PLIST_TARGET_DIR="$HOME/Library/LaunchAgents"
PLIST_TARGET="$PLIST_TARGET_DIR/com.felipeveas.dotctl.statusapp.plist"
APP_BIN="${STATUS_APP_BIN:-$ROOT_DIR/bin/StatusApp.app/Contents/MacOS/StatusApp}"

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "error: scripts/install-launchagent-macos.sh must run on macOS"
  exit 1
fi

mkdir -p "$PLIST_TARGET_DIR"
sed "s|/Applications/StatusApp.app/Contents/MacOS/StatusApp|$APP_BIN|g" "$PLIST_SOURCE" > "$PLIST_TARGET"
echo "Installed LaunchAgent plist: $PLIST_TARGET"
echo "Configured binary path: $APP_BIN"

if command -v launchctl >/dev/null 2>&1; then
  launchctl unload "$PLIST_TARGET" >/dev/null 2>&1 || true
  launchctl load "$PLIST_TARGET"
  echo "Loaded LaunchAgent: com.felipeveas.dotctl.statusapp"
else
  echo "warning: launchctl not found, skipping load step"
fi
