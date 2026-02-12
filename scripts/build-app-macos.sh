#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "error: scripts/build-app-macos.sh must run on macOS"
  exit 1
fi

if ! command -v lipo >/dev/null 2>&1; then
  echo "error: lipo not found (install Xcode Command Line Tools)"
  exit 1
fi

if ! command -v swiftc >/dev/null 2>&1; then
  echo "error: swiftc not found (install Xcode Command Line Tools)"
  exit 1
fi

VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
APP_SRC_DIR="$ROOT_DIR/mac/StatusApp/StatusApp"
APP_BUNDLE="${APP_BUNDLE:-$ROOT_DIR/bin/StatusApp.app}"
BUILD_DIR="$ROOT_DIR/mac/StatusApp/build"
DOTCTL_BIN_DIR="$ROOT_DIR/mac/StatusApp/bin"

mkdir -p "$BUILD_DIR" "$DOTCTL_BIN_DIR"

export GOCACHE="${GOCACHE:-${TMPDIR:-/tmp}/dotctl-gocache}"
export CLANG_MODULE_CACHE_PATH="${CLANG_MODULE_CACHE_PATH:-$BUILD_DIR/clang-modules}"
export SWIFT_MODULECACHE_PATH="${SWIFT_MODULECACHE_PATH:-$BUILD_DIR/swift-modules}"
mkdir -p "$GOCACHE" "$CLANG_MODULE_CACHE_PATH" "$SWIFT_MODULECACHE_PATH"

DOTCTL_ARM64="$BUILD_DIR/dotctl-arm64"
DOTCTL_AMD64="$BUILD_DIR/dotctl-amd64"
DOTCTL_UNIVERSAL="$DOTCTL_BIN_DIR/dotctl"

echo "Building dotctl universal binary..."
GOOS=darwin GOARCH=arm64 go build \
  -trimpath \
  -ldflags "-X github.com/felipe-veas/dotctl/internal/version.Version=$VERSION" \
  -o "$DOTCTL_ARM64" \
  ./cmd/dotctl
GOOS=darwin GOARCH=amd64 go build \
  -trimpath \
  -ldflags "-X github.com/felipe-veas/dotctl/internal/version.Version=$VERSION" \
  -o "$DOTCTL_AMD64" \
  ./cmd/dotctl
lipo -create "$DOTCTL_ARM64" "$DOTCTL_AMD64" -output "$DOTCTL_UNIVERSAL"
chmod +x "$DOTCTL_UNIVERSAL"

echo "Compiling StatusApp menubar binary..."
STATUS_BIN="$BUILD_DIR/StatusApp"
swiftc \
  "$APP_SRC_DIR/AppDelegate.swift" \
  "$APP_SRC_DIR/StatusBarController.swift" \
  "$APP_SRC_DIR/DotctlBridge.swift" \
  -framework AppKit \
  -o "$STATUS_BIN"

echo "Assembling app bundle..."
mkdir -p "$APP_BUNDLE/Contents/MacOS" "$APP_BUNDLE/Contents/Resources"
cp "$STATUS_BIN" "$APP_BUNDLE/Contents/MacOS/StatusApp"
cp "$APP_SRC_DIR/Info.plist" "$APP_BUNDLE/Contents/Info.plist"
cp "$DOTCTL_UNIVERSAL" "$APP_BUNDLE/Contents/Resources/dotctl"

chmod +x "$APP_BUNDLE/Contents/MacOS/StatusApp" "$APP_BUNDLE/Contents/Resources/dotctl"

echo "App ready at: $APP_BUNDLE"
