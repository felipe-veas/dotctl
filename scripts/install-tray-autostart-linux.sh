#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODE="${1:-desktop}" # desktop | systemd | both

if [[ "$(uname -s)" != "Linux" ]]; then
  echo "error: scripts/install-tray-autostart-linux.sh must run on Linux"
  exit 1
fi

install_desktop() {
  local target_dir="${XDG_CONFIG_HOME:-$HOME/.config}/autostart"
  mkdir -p "$target_dir"
  cp "$ROOT_DIR/linux/tray/autostart/dotctl-tray.desktop" "$target_dir/"
  echo "Installed autostart desktop entry: $target_dir/dotctl-tray.desktop"
}

install_systemd() {
  local target_dir="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
  mkdir -p "$target_dir"
  cp "$ROOT_DIR/linux/tray/systemd/dotctl-tray.service" "$target_dir/"
  echo "Installed user service: $target_dir/dotctl-tray.service"

  if command -v systemctl >/dev/null 2>&1; then
    systemctl --user daemon-reload
    systemctl --user enable --now dotctl-tray.service
    echo "Enabled dotctl-tray.service for current user"
  else
    echo "warning: systemctl not found, skipped enable step"
  fi
}

case "$MODE" in
  desktop)
    install_desktop
    ;;
  systemd)
    install_systemd
    ;;
  both)
    install_desktop
    install_systemd
    ;;
  *)
    echo "usage: $0 [desktop|systemd|both]"
    exit 1
    ;;
esac
