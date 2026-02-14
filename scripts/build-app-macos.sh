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

if ! command -v codesign >/dev/null 2>&1; then
  echo "error: codesign not found (install Xcode Command Line Tools)"
  exit 1
fi

if ! command -v sips >/dev/null 2>&1; then
  echo "error: sips not found (install Xcode Command Line Tools)"
  exit 1
fi

if ! command -v iconutil >/dev/null 2>&1; then
  echo "error: iconutil not found (install Xcode Command Line Tools)"
  exit 1
fi

VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
APP_SRC_DIR="$ROOT_DIR/mac/DotCtl/DotCtl"
APP_BUNDLE="${APP_BUNDLE:-$ROOT_DIR/bin/DotCtl.app}"
BUILD_DIR="$ROOT_DIR/mac/DotCtl/build"
DOTCTL_BIN_DIR="$ROOT_DIR/mac/DotCtl/bin"
ICON_EXPORT_SRC="$ROOT_DIR/mac/DotCtl/tools/export-app-icon.swift"

mkdir -p "$BUILD_DIR" "$DOTCTL_BIN_DIR"

export GOCACHE="${GOCACHE:-${TMPDIR:-/tmp}/dotctl-gocache}"
export CLANG_MODULE_CACHE_PATH="${CLANG_MODULE_CACHE_PATH:-$BUILD_DIR/clang-modules}"
export SWIFT_MODULECACHE_PATH="${SWIFT_MODULECACHE_PATH:-$BUILD_DIR/swift-modules}"
mkdir -p "$GOCACHE" "$CLANG_MODULE_CACHE_PATH" "$SWIFT_MODULECACHE_PATH"

DOTCTL_ARM64="$BUILD_DIR/dotctl-arm64"
DOTCTL_AMD64="$BUILD_DIR/dotctl-amd64"
DOTCTL_UNIVERSAL="$DOTCTL_BIN_DIR/dotctl"
ICON_EXPORT_BIN="$BUILD_DIR/dotctl-icon-export"
ICON_BASE_PNG="$BUILD_DIR/DotCtl-1024.png"
ICONSET_DIR="$BUILD_DIR/DotCtl.iconset"
ICON_ICNS="$BUILD_DIR/DotCtl.icns"

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

echo "Generating DotCtl app icon..."
swiftc \
  "$APP_SRC_DIR/DotCtlIcon.swift" \
  "$ICON_EXPORT_SRC" \
  -framework AppKit \
  -o "$ICON_EXPORT_BIN"
"$ICON_EXPORT_BIN" "$ICON_BASE_PNG"
rm -rf "$ICONSET_DIR"
mkdir -p "$ICONSET_DIR"
for size in 16 32 128 256 512; do
  sips -z "$size" "$size" "$ICON_BASE_PNG" --out "$ICONSET_DIR/icon_${size}x${size}.png" >/dev/null
done
for size in 16 32 128 256 512; do
  scaled=$((size * 2))
  sips -z "$scaled" "$scaled" "$ICON_BASE_PNG" --out "$ICONSET_DIR/icon_${size}x${size}@2x.png" >/dev/null
done
iconutil -c icns "$ICONSET_DIR" -o "$ICON_ICNS"

echo "Compiling DotCtl menubar binary..."
APP_BIN="$BUILD_DIR/DotCtl"
swiftc \
  "$APP_SRC_DIR/AppDelegate.swift" \
  "$APP_SRC_DIR/DotCtlIcon.swift" \
  "$APP_SRC_DIR/StatusBarController.swift" \
  "$APP_SRC_DIR/DotctlBridge.swift" \
  -framework AppKit \
  -parse-as-library \
  -o "$APP_BIN"

echo "Assembling app bundle..."
rm -rf "$APP_BUNDLE"
mkdir -p "$APP_BUNDLE/Contents/MacOS" "$APP_BUNDLE/Contents/Resources"
cp "$APP_BIN" "$APP_BUNDLE/Contents/MacOS/DotCtl"
cp "$APP_SRC_DIR/Info.plist" "$APP_BUNDLE/Contents/Info.plist"
cp "$DOTCTL_UNIVERSAL" "$APP_BUNDLE/Contents/Resources/dotctl"
cp "$ICON_ICNS" "$APP_BUNDLE/Contents/Resources/DotCtl.icns"

chmod +x "$APP_BUNDLE/Contents/MacOS/DotCtl" "$APP_BUNDLE/Contents/Resources/dotctl"
codesign --force --deep --sign - "$APP_BUNDLE" >/dev/null

echo "App ready at: $APP_BUNDLE"
