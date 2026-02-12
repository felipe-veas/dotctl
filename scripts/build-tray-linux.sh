#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if [[ "$(uname -s)" != "Linux" ]]; then
  echo "error: scripts/build-tray-linux.sh must run on Linux"
  exit 1
fi

if ! command -v pkg-config >/dev/null 2>&1; then
  echo "error: pkg-config is required (install package: pkg-config)"
  exit 1
fi

if ! pkg-config --exists ayatana-appindicator3-0.1 && ! pkg-config --exists appindicator3-0.1; then
  cat <<'MSG'
error: appindicator headers not found.
Install one of:
  - libayatana-appindicator3-dev
  - libappindicator3-dev
MSG
  exit 1
fi

VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
OUT_BIN="${OUT_BIN:-$ROOT_DIR/bin/dotctl-tray}"

mkdir -p "$(dirname "$OUT_BIN")"

export GOCACHE="${GOCACHE:-${TMPDIR:-/tmp}/dotctl-gocache}"
mkdir -p "$GOCACHE"

echo "Building Linux tray app -> $OUT_BIN"
CGO_ENABLED=1 go build \
  -tags tray \
  -trimpath \
  -ldflags "-X github.com/felipe-veas/dotctl/internal/version.Version=$VERSION" \
  -o "$OUT_BIN" \
  ./linux/tray

echo "Build complete: $OUT_BIN"
