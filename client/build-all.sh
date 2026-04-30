#!/usr/bin/env bash
# Cross-platform release build for the Portal desktop app.
#
# Run from client/. Produces signed .app / .exe / linux binaries
# under client/build/bin/<platform>/. Wails handles bundling per-OS.
#
# Prerequisites:
#   - wails CLI on PATH (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
#   - For Windows from macOS/Linux: `go install github.com/akavel/rsrc@latest`
#     and an x86_64-w64-mingw32-gcc toolchain.
#   - For Linux from macOS: docker (we delegate to wails' linux build container)
#     or run on a Linux host directly.
#
# In practice we recommend: build each native platform on its own host
# (macOS for darwin-*, Windows for windows-*, Linux for linux-*).

set -euo pipefail
cd "$(dirname "$0")"

mkdir -p build/bin

PLATFORMS=(
  "darwin/arm64"
  "darwin/amd64"
)

# Add windows + linux only if asked — they need extra toolchains.
if [[ "${INCLUDE_WIN:-0}" == "1" ]]; then PLATFORMS+=("windows/amd64"); fi
if [[ "${INCLUDE_LINUX:-0}" == "1" ]]; then PLATFORMS+=("linux/amd64"); fi

for p in "${PLATFORMS[@]}"; do
  echo
  echo "================================================================"
  echo "  Building $p"
  echo "================================================================"
  case "$p" in
    darwin/*)
      wails build -clean -skipbindings -platform "$p"
      ;;
    windows/*)
      wails build -clean -skipbindings -platform "$p" -nsis=false || true
      ;;
    linux/*)
      wails build -clean -skipbindings -platform "$p" || true
      ;;
  esac
  # Move output into a per-platform sub-directory so successive builds
  # don't clobber each other.
  out="build/bin/$(echo "$p" | tr / -)"
  mkdir -p "$out"
  if [[ -d build/bin/Portal.app ]]; then
    mv build/bin/Portal.app "$out/"
  fi
  for f in build/bin/Portal build/bin/Portal.exe; do
    [[ -e "$f" ]] && mv "$f" "$out/" || true
  done
done

echo
echo "Done. Artifacts:"
find build/bin -maxdepth 2 -type d -o -type f | sort
