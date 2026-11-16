#!/bin/sh
# Cross-compiles the release archives into dist/: one tar.gz per target (zip
# on Windows) and a SHA256SUMS file.
#
# Usage: scripts/dist.sh <version>
#
# "just dist" runs this, and so does the release workflow.
set -eu

version=${1:?usage: scripts/dist.sh <version>}
cd "$(dirname "$0")/.."

# No macOS AppleDouble (._*) files in the archives.
export COPYFILE_DISABLE=1

ldflags="-s -w -X github.com/ailequal/ffcc-gci/internal/cli.Version=$version"

rm -rf dist
mkdir dist

for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do
  os=${target%/*}
  arch=${target#*/}
  name="ffcc-gci-$version-$os-$arch"
  ext=""
  [ "$os" = windows ] && ext=".exe"

  # The documents keep their place, so the links in README.md still work.
  mkdir -p "dist/$name/docs"
  CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath -ldflags "$ldflags" \
    -o "dist/$name/ffcc-gci$ext" ./cmd/ffcc-gci
  cp README.md LICENSE CHANGELOG.md "dist/$name/"
  cp docs/GUIDE.md docs/USAGE.md docs/SAVE_FORMAT.md "dist/$name/docs/"

  if [ "$os" = windows ]; then
    (cd dist && zip -qr "$name.zip" "$name")
  else
    tar --no-xattrs -czf "dist/$name.tar.gz" -C dist "$name"
  fi
  rm -r "dist/$name"
done

(
  cd dist
  if command -v sha256sum >/dev/null; then
    sha256sum ffcc-gci-* >SHA256SUMS
  else
    shasum -a 256 ffcc-gci-* >SHA256SUMS
  fi
)
ls -l dist
