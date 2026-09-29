#!/usr/bin/env bash
#
# Builds and packages XinText on macOS or Linux, run ON the target platform
# (or inside the wails-cross Docker container for cross builds).
#
#   macOS:  .app bundle (+ optional DMG) and a portable .tar.gz
#   Linux:  AppImage / deb / rpm (via `wails3 task linux:package`) plus a
#           portable .tar.gz
#
# Bundled tools are picked up automatically by the Go backend from locations
# next to the executable:
#   pandoc/<pandoc>     -> HTML/DOCX/TXT export
#
# PDF export needs no external engine: it is rendered by the system browser
# engine (Microsoft Edge on Windows).
#
# Environment:
#   PANDOC_BIN  path to the target-platform pandoc binary to bundle (optional)
#   SKIP_BUILD=1 reuse the existing binary
#   DMG=1       (macOS) also produce a DMG
#
# Linux build needs the system WebKit2GTK libraries, e.g. on Debian/Ubuntu:
#   sudo apt install libgtk-3-dev libwebkit2gtk-4.1-dev
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

command -v wails3 >/dev/null 2>&1 || {
  echo "error: wails3 not found on PATH (install the Wails v3 CLI)" >&2
  exit 1
}

VERSION="$(sed -n 's/^[[:space:]]*version:[[:space:]]*"\([^"]*\)".*/\1/p' build/config.yml | head -n1)"
VERSION="${VERSION:-0.0.0}"
OS="$(uname -s)"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
esac

if [[ "${SKIP_BUILD:-0}" != "1" ]]; then
  echo "==> Production build"
  wails3 build
fi

bundle_tools() {  # $1 = directory containing the executable
  local bindir="$1"
  if [[ -n "${PANDOC_BIN:-}" && -f "$PANDOC_BIN" ]]; then
    mkdir -p "$bindir/pandoc"
    cp "$PANDOC_BIN" "$bindir/pandoc/pandoc"
    chmod +x "$bindir/pandoc/pandoc"
    echo "==> Bundled pandoc from $PANDOC_BIN"
  else
    echo "warning: no pandoc bundled (set PANDOC_BIN); HTML/DOCX/TXT export needs pandoc on PATH" >&2
  fi
}

make_targz() {  # $1 = staging dir name, $2 = top folder inside
  local stage="$1" top="$2"
  local out="bin/XinText-$VERSION-$OS-$ARCH-portable.tar.gz"
  tar -czf "$out" -C "$stage" "$top"
  echo "==> Portable archive: $out"
}

case "$OS" in
  Darwin)
    echo "==> Packaging .app bundle"
    wails3 task darwin:package
    APP="bin/XinText.app"
    [[ -d "$APP" ]] || { echo "error: $APP not produced" >&2; exit 1; }
    # Binary lives in Contents/MacOS; bundled tools sit next to it.
    bundle_tools "$APP/Contents/MacOS"
    # Portable tarball of the bundle.
    rm -rf "bin/dmg-stage"; mkdir -p "bin/dmg-stage"
    cp -R "$APP" "bin/dmg-stage/"
    make_targz "bin/dmg-stage" "XinText.app"
    rm -rf "bin/dmg-stage"
    if [[ "${DMG:-0}" == "1" ]]; then
      echo "==> Packaging DMG"
      wails3 task darwin:package:dmg
    fi
    ;;
  Linux)
    echo "==> Packaging AppImage / deb / rpm"
    wails3 task linux:package || {
      echo "error: linux packaging failed; ensure libwebkit2gtk-4.1-dev and libgtk-3-dev are installed" >&2
      exit 1
    }
    # Portable tarball: binary + bundled tools.
    BIN="bin/XinText"
    [[ -f "$BIN" ]] || { echo "error: $BIN not produced" >&2; exit 1; }
    TOP="XinText-$VERSION-linux-$ARCH"
    STAGE="bin/tar-stage/$TOP"
    rm -rf "bin/tar-stage"; mkdir -p "$STAGE"
    cp "$BIN" "$STAGE/XinText"; chmod +x "$STAGE/XinText"
    bundle_tools "$STAGE"
    make_targz "bin/tar-stage" "$TOP"
    rm -rf "bin/tar-stage"
    ;;
  *)
    echo "error: unsupported platform $OS" >&2
    exit 1
    ;;
esac

echo "==> Done. Artifacts in bin/"
ls -lh bin/XinText-"$VERSION"-* 2>/dev/null || true
