#!/usr/bin/env bash
# Cross-compile raptor for the full release matrix into dist/.
#
# Expects the frontend to be pre-built into internal/webui/dist (CI runs
# `cd web && npm ci && npm run build` first). VERSION is injected via ldflags.
set -euo pipefail

VERSION="${VERSION:-dev}"
PKG="github.com/t0mer/raptor/internal/version.Version"
LDFLAGS="-s -w -X ${PKG}=${VERSION}"
OUT="dist"

rm -rf "$OUT"
mkdir -p "$OUT"

# OS/arch/arm-variant triples.
TARGETS=(
	"linux amd64 "
	"linux arm64 "
	"linux arm 7"
	"linux arm 6"
	"linux 386 "
	"darwin amd64 "
	"darwin arm64 "
	"windows amd64 "
	"windows arm64 "
)

for t in "${TARGETS[@]}"; do
	read -r GOOS GOARCH GOARM <<<"$t"
	name="raptor-${VERSION}-${GOOS}-${GOARCH}"
	[ -n "$GOARM" ] && name="${name}v${GOARM}"
	ext=""
	[ "$GOOS" = "windows" ] && ext=".exe"

	echo "building ${name}${ext}"
	CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" GOARM="$GOARM" \
		go build -trimpath -ldflags "$LDFLAGS" -o "${OUT}/${name}${ext}" ./cmd/raptor/
done

echo "done -> ${OUT}/"
