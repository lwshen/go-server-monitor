//go:build !embed

package web

import "io/fs"

// Assets reports that no SPA is embedded in this (development / default) build —
// the frontend is served by the Vite dev server instead. Release images build
// with `-tags embed` (see spa_embed.go, Dockerfile, Makefile `release`).
func Assets() (fs.FS, bool) { return nil, false }
