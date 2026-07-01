#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# scripts/build.sh — cross-compile server + probe binaries (P0 skeleton template).
#
# Mirrors `make build-all`. Builds for linux/darwin x amd64/arm64 into ./bin/.
#
# Authority: requirements/12-build-plan.md (P8 REQ-DEPLOY-08),
#            requirements/10-deployment-ops.md (REQ-OPS-01 cross-platform).
#
# Usage:  ./scripts/build.sh [VERSION]
#   VERSION defaults to "dev" and is stamped into the binary (main.Version).
#
# Note: CGO is disabled because modernc.org/sqlite is pure Go — the binaries
# are fully static and cross-compile without a C toolchain.
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

cd "$(dirname "$0")/.."

VERSION="${1:-dev}"
LDFLAGS="-s -w -X main.Version=${VERSION}"
BIN_DIR="bin"
PLATFORMS=("linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64")

mkdir -p "${BIN_DIR}"

# Build the SPA first so the server can embed it (-tags embed -> //go:embed dist).
echo "==> building web SPA"
( cd web && { [ -d node_modules ] || npm ci; } && npm run build )

echo "==> building go-server-monitor (version=${VERSION})"
for platform in "${PLATFORMS[@]}"; do
  os="${platform%/*}"
  arch="${platform#*/}"
  echo "    server  ${os}/${arch} (embedded SPA)"
  CGO_ENABLED=0 GOOS="${os}" GOARCH="${arch}" \
    go build -tags embed -trimpath -ldflags "${LDFLAGS}" \
    -o "${BIN_DIR}/server-${os}-${arch}" ./cmd/server
  echo "    probe   ${os}/${arch}"
  CGO_ENABLED=0 GOOS="${os}" GOARCH="${arch}" \
    go build -trimpath -ldflags "${LDFLAGS}" \
    -o "${BIN_DIR}/probe-${os}-${arch}" ./cmd/probe
done

echo "==> done. artifacts in ${BIN_DIR}/:"
ls -lh "${BIN_DIR}/"
