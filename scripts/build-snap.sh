#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SNAP_DIR="${ROOT_DIR}/packaging/snap"

if ! command -v snapcraft >/dev/null 2>&1; then
  echo "snapcraft not found. Install it first (e.g. sudo snap install snapcraft --classic)." >&2
  exit 1
fi

echo "Building snap package from ${SNAP_DIR}/snapcraft.yaml"
snapcraft --destructive-mode --project-dir "${SNAP_DIR}"
