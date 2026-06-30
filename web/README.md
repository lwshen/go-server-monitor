# Server Monitor — Frontend SPA

Vue 3 + Vite + TypeScript single-page app for the self-hosted server-monitoring system.

> **Status: P0 skeleton.** This is the project frame only. Views, components, services,
> and stores are placeholders. Real data logic lands in **P5** (dashboard / detail) and
> **P6** (admin). See `requirements/12-build-plan.md`.

## Prerequisites

- Node.js 18+
- npm

## Install

```bash
npm install
```

> Note: not run during P0 scaffolding. Run it before `dev`/`build`.

## Develop

```bash
npm run dev
```

Starts Vite on http://localhost:5173. `/api` and `/ws` requests are proxied to the
Go backend on http://localhost:8080 (see `vite.config.ts`).

## Build

```bash
npm run build      # type-check (vue-tsc) + production build to dist/
npm run preview    # preview the dist/ build locally
npm run type-check # vue-tsc only, no emit
```

Build output goes to `dist/`, later embedded into the Go binary (P8).

## Configuration

| Env var             | Default                  | Description                          |
| ------------------- | ------------------------ | ------------------------------------ |
| `VITE_API_BASE_URL` | `http://localhost:8080`  | Base URL for backend API calls.      |

## Layout

```
web/
  index.html
  vite.config.ts
  src/
    main.ts              # app bootstrap (router + pinia + i18n)
    App.vue              # root shell (nav, theme/lang toggles — placeholders)
    router/index.ts      # / , /server/:id , /admin
    pages/               # DashboardPage, ServerDetailPage, AdminPage
    components/          # ServerCard, ServerTable, MetricsChart, WorldMap
    services/            # api.ts (axios), ws.ts (WebSocket manager)
    stores/              # servers.ts (Pinia)
    i18n/                # index.ts, zh.ts, en.ts
```
