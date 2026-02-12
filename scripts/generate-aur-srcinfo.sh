#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
AUR_DIR="${ROOT_DIR}/packaging/aur"

if ! command -v makepkg >/dev/null 2>&1; then
  echo "makepkg not found. Run this script on Arch/Manjaro or in an Arch container." >&2
  exit 1
fi

cd "${AUR_DIR}"
makepkg --printsrcinfo > .SRCINFO
echo "Updated ${AUR_DIR}/.SRCINFO"
