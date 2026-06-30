# syntax=docker/dockerfile:1
# ─────────────────────────────────────────────────────────────────────────────
# go-server-monitor — multi-stage container image (P0 skeleton template).
#
# Authority: requirements/10-deployment-ops.md (REQ-OPS-01/02),
#            requirements/12-build-plan.md (P0 deliverable 5, P8 REQ-DEPLOY-05).
#
# Goal: a small (<50MB-ish) image. The server is a single static Go binary
# (CGO_ENABLED=0 — modernc.org/sqlite is pure Go, so no cgo / libc needed),
# bundled with the built Vue SPA in /app/web/dist.
#
# NOTE(P8): once REQ-DEPLOY-03 lands, the SPA in web/dist is expected to be
# embedded into the Go binary via //go:embed. Until then the runtime stage
# copies web/dist alongside the binary and Caddy / the server serves it from
# the filesystem. This Dockerfile is written to work either way.
# ─────────────────────────────────────────────────────────────────────────────

# ── Stage 1: build the Vue 3 SPA → web/dist ─────────────────────────────────
FROM node:22-alpine AS web-builder
WORKDIR /web

# Install deps first (better layer caching). package-lock.json may not exist
# yet in P0; `npm ci` requires it, so fall back to `npm install` when absent.
COPY web/package.json web/package-lock.json* ./
RUN if [ -f package-lock.json ]; then npm ci; else npm install; fi

# Build the SPA. `npm run build` runs `vue-tsc --noEmit && vite build`
# (see web/package.json) and emits to web/dist (vite.config.ts outDir=dist).
COPY web/ ./
RUN npm run build

# ── Stage 2: build the Go server binary (static, pure Go) ───────────────────
FROM golang:1.24-alpine AS go-builder
WORKDIR /src

# git is occasionally needed for module resolution / VCS stamping.
RUN apk add --no-cache git

# Cache modules independently of source.
COPY go.mod go.sum* ./
RUN go mod download

# Build the static binary. CGO disabled → fully static, runs on bare alpine.
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags "-s -w -X main.Version=${VERSION}" \
    -o /out/server ./cmd/server

# ── Stage 3: minimal runtime ────────────────────────────────────────────────
FROM alpine:latest AS runtime

# ca-certificates: outbound HTTPS (Telegram/webhook notifications, P6).
# curl: container HEALTHCHECK against /health.
# tzdata: correct local-time rendering for alert messages (TZ env).
RUN apk add --no-cache ca-certificates curl tzdata

WORKDIR /app

# Server binary + built SPA. When go:embed lands (P8) the web/dist copy
# becomes optional, but keeping it is harmless.
COPY --from=go-builder /out/server /app/server
COPY --from=web-builder /web/dist /app/web/dist

# Persisted data (SQLite db + WAL/SHM). Mounted as a volume in compose.
RUN mkdir -p /data

EXPOSE 8080

# Frozen public health endpoint is /health (REQ-RES-00), not /api/health.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -fsS http://localhost:8080/health || exit 1

# TODO(P0/P8): the server reads config from env (.env) and/or config.yaml.
CMD ["/app/server"]
