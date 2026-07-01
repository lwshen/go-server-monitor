//go:build embed

// Package web optionally embeds the built Vue SPA (web/dist) into the binary.
//
// This file is compiled ONLY with `-tags embed` (release builds), because
// //go:embed of web/dist requires the frontend to have been built first
// (`npm run build`). Default `go build ./...` uses spa_noembed.go instead, so
// development and CI compile without needing dist to exist.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// Assets returns the embedded SPA filesystem (rooted at dist) and true.
func Assets() (fs.FS, bool) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, false
	}
	return sub, true
}
