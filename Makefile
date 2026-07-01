# ─────────────────────────────────────────────────────────────────────────────
# go-server-monitor — Makefile (P0 skeleton).
#
# Authority: requirements/12-build-plan.md (P0/P8 deliverables),
#            requirements/01-architecture.md (REQ-ARCH-06 Makefile in layout).
#
# Common targets:
#   make build        build the server binary into bin/server (host)
#   make build-probe  build the probe binary into bin/probe (host)
#   make build-web    build the Vue SPA into web/dist (auto npm ci if needed)
#   make build-all    build EVERYTHING for the host: web + server + probe
#   make release      cross-compile server+probe for all platforms (+ web once)
#   make run          go run the server
#   make test/vet/fmt/tidy   standard Go hygiene
#   make install-web  install frontend deps (npm ci)
#   make docker-build / docker-up / docker-down   container workflow
#   make clean        remove build artifacts
# ─────────────────────────────────────────────────────────────────────────────

# Version stamped into the binary via -ldflags. Override: make build VERSION=1.2.3
VERSION ?= dev
LDFLAGS := -s -w -X main.Version=$(VERSION)

BIN_DIR     := bin
SERVER_PKG  := ./cmd/server
PROBE_PKG   := ./cmd/probe
# CGO disabled: modernc.org/sqlite is pure Go → static, cross-compile friendly.
GOBUILD       := CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)"
# `-tags embed` bundles web/dist into the binary (//go:embed); requires a prior
# `build-web`. Dev builds omit it so `go build` works without dist.
GOBUILD_EMBED := CGO_ENABLED=0 go build -tags embed -trimpath -ldflags "$(LDFLAGS)"

# Cross-compile matrix for the release target.
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64

.PHONY: all build build-probe build-web build-embed build-all release run tidy test vet fmt \
        install-web docker-build docker-up docker-down clean

# Bare `make` = quick host server build (fast inner loop).
# Use `make build-all` for the whole project (web + both binaries).
all: build

## build: compile the server binary into bin/server (host)
build:
	$(GOBUILD) -o $(BIN_DIR)/server $(SERVER_PKG)

## build-probe: compile the probe binary into bin/probe (host)
build-probe:
	$(GOBUILD) -o $(BIN_DIR)/probe $(PROBE_PKG)

## build-embed: host server binary with the SPA embedded (needs a built web/dist)
build-embed: build-web
	$(GOBUILD_EMBED) -o $(BIN_DIR)/server $(SERVER_PKG)

## build-all: build a runnable single binary (SPA-embedded server) + probe
build-all: build-embed build-probe
	@echo "done -> $(BIN_DIR)/server (SPA embedded) + $(BIN_DIR)/probe"

## release: cross-compile the SPA-embedded server + probe for all platforms
release: build-web
	@mkdir -p $(BIN_DIR)
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; arch=$${platform#*/}; \
		echo "building server $$os/$$arch (embedded SPA)"; \
		GOOS=$$os GOARCH=$$arch $(GOBUILD_EMBED) -o $(BIN_DIR)/server-$$os-$$arch $(SERVER_PKG); \
		echo "building probe  $$os/$$arch"; \
		GOOS=$$os GOARCH=$$arch $(GOBUILD) -o $(BIN_DIR)/probe-$$os-$$arch $(PROBE_PKG); \
	done
	@echo "done -> $(BIN_DIR)/"

## run: run the server from source
run:
	go run $(SERVER_PKG)

## tidy: sync go.mod / go.sum
tidy:
	go mod tidy

## test: run the test suite
test:
	go test ./...

## vet: run go vet static checks
vet:
	go vet ./...

## fmt: format all Go source
fmt:
	go fmt ./...

## install-web: install frontend dependencies (clean install)
install-web:
	cd web && npm ci

## build-web: build the Vue SPA into web/dist (auto-installs deps if missing)
build-web:
	cd web && { [ -d node_modules ] || npm ci; } && npm run build

## docker-build: build the container image
docker-build:
	docker build --build-arg VERSION=$(VERSION) -t go-server-monitor:$(VERSION) .

## docker-up: start the stack via docker compose
docker-up:
	docker compose up -d

## docker-down: stop the stack
docker-down:
	docker compose down

## clean: remove build artifacts
clean:
	rm -rf $(BIN_DIR) web/dist
