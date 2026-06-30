# go-server-monitor

A lightweight, fully self-hosted multi-server monitoring system. A Go reimplementation
of CF-Server-Monitor (originally Cloudflare Workers + D1) fused with the
ServerStatus-Rust data structures, replacing the cloud runtime with a single Go
process: SQLite for storage, an in-process goroutine WebSocket Hub for realtime
push, a Vue 3 SPA dashboard, and Caddy for automatic HTTPS. Probes collect
CPU / memory / disk / network / connection-count / multi-line (telecom, unicom,
mobile, Baidu) network-quality metrics every ~5s and `POST /report` them over
HTTPS; the backend authenticates, stores every sample (180-day retention by
default), and broadcasts live updates to the dashboard. It runs comfortably on a
128–256MB VPS with no external database, message queue, or cloud dependency.

> Status: **P0 — project skeleton.** Most of the system is stubbed; see the build
> phase table below. The full specification lives in [`requirements/`](requirements/).

## Tech stack

- **Backend**: Go 1.24 (single static binary, `CGO_ENABLED=0`)
- **Storage**: SQLite via `modernc.org/sqlite` (pure Go, WAL mode)
- **Realtime**: `coder/websocket` + in-process goroutine broadcast Hub
- **Scheduling**: `robfig/cron` (offline detection, retention cleanup, reminders)
- **Auth**: bcrypt password hashing + JWT (HS256, 7-day) for admin; constant-time
  shared-secret check for probe uploads
- **Frontend**: Vue 3 + Vite SPA (Pinia, Vue Router, vue-i18n, Chart.js, Leaflet)
- **Deploy**: single binary or docker-compose; Caddy auto-HTTPS reverse proxy

## Quick start (development)

```bash
# 1. Configure secrets
cp .env.example .env        # then edit: set API_SECRET and ADMIN_PASSWORD

# 2. Run the backend (defaults to :8080, db at ./data/metrics.db)
go run ./cmd/server
#   or: make run

# 3. Run the frontend dev server (proxies /api and /ws to :8080)
cd web && npm install && npm run dev   # http://localhost:5173
```

Health check (frozen public endpoint): `curl http://localhost:8080/health`

## Build

```bash
make build            # -> bin/server (host)
make build-probe      # -> bin/probe (host)
make build-web        # build the Vue SPA into web/dist (auto npm ci if needed)
make build-all        # build everything for the host: web + server + probe
make release          # cross-compile server + probe for linux/darwin x amd64/arm64 (+ web)

./scripts/build.sh [VERSION]   # same cross-compile matrix as `make release`
```

## Docker / docker-compose

```bash
# Build the image
make docker-build           # or: docker build -t go-server-monitor:dev .

# Bring up the full stack (app + Caddy). Requires .env with API_SECRET / ADMIN_PASSWORD.
make docker-up              # or: docker compose up -d
make docker-down            # tear down

# One-command deploy with health check
./scripts/deploy.sh [API_SECRET] [ADMIN_PASSWORD]
```

The `app` service exposes the Go server on `:8080`; the `caddy` service fronts it
on `80/443` with automatic HTTPS. Set `DOMAIN=your.hostname` in `.env` to obtain a
Let's Encrypt certificate (defaults to `localhost` for local runs). The SQLite
database persists under `./data`.

## Frozen conventions (quick reference)

- **Units**: memory/swap/disk aggregate = MiB; `disks[]` detail = bytes; net
  cumulative = bytes; net speed = B/s; timestamps = **Unix seconds**; ping = ms;
  cpu/loss = % (0–100).
- **Sentinel**: an unmeasured numeric is `-1` on the wire → SQL `NULL` in the DB →
  `—` in the UI (chiefly `ping_*` / `loss_*`).
- **Endpoints** (frozen): `POST /report`, `GET /api/config`, `GET /api/servers`,
  `GET /api/server?id=`, `GET /api/history?id=&range=`, `GET /ws?subscribe=`,
  `POST /api/admin/login`, `POST /api/admin/servers` (+ `/add` `/edit` `/delete`
  `/reorder`), `GET|POST /api/admin/settings`, `POST /api/admin/db/rebuild`,
  `GET /health`.

See [`requirements/CONVENTIONS.md`](requirements/CONVENTIONS.md) and
[`requirements/14-resolved-decisions.md`](requirements/14-resolved-decisions.md)
for the authoritative, frozen definitions.

## Build phases (P0–P8)

| Phase | Focus | Key deliverables |
|-------|-------|------------------|
| **P0** | Project skeleton | Go module, directory layout, config loading, health check |
| **P1** | Database | SQLite three tables, PRAGMA tuning, indexes, migration + init |
| **P2** | Report ingest | `POST /report`, secret auth, StatReport parse → store |
| **P3** | Realtime | goroutine Hub, `GET /ws`, hello/update/batchUpdate, ingest→broadcast |
| **— MVP boundary (P0–P3): end-to-end probe → SQLite → WebSocket → frontend —** | | |
| **P4** | Frontend UI | Vue 3 SPA, dashboard/detail, realtime subscribe + polling fallback, map & charts |
| **P5** | Admin backend | JWT login, server CRUD/reorder, settings, install-command generation |
| **P6** | Scheduling & alerts | cron offline detection / expiry reminder / retention cleanup; Telegram/Webhook |
| **P7** | Deployment | single binary, docker-compose, Caddy, backups, graceful shutdown, CI/CD |
| **P8** | Polish | non-functional acceptance, i18n/theme, observability, security baseline |

> Note: the phase numbering above follows `requirements/README.md`. The detailed
> per-stage acceptance criteria live in
> [`requirements/12-build-plan.md`](requirements/12-build-plan.md) (which also
> splits deployment across its own P0/P8 deliverables).

## Full specification

The complete, authoritative spec is in [`requirements/`](requirements/). Start with
`CONVENTIONS.md` (highest authority), then `14-resolved-decisions.md` (frozen
decisions), `00-overview.md`, and `01-architecture.md`. The single source of truth
for the report contract is `requirements/report-types.ts`.
