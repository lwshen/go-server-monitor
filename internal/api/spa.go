package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// apiPrefixes are the server-owned path prefixes; an unmatched request under one
// of these is a real 404 (JSON), not a SPA client route.
var apiPrefixes = []string{"/api", "/report", "/ws", "/health"}

func isAPIPath(p string) bool {
	for _, pre := range apiPrefixes {
		if p == pre || strings.HasPrefix(p, pre+"/") {
			return true
		}
	}
	return false
}

// registerSPA serves the embedded Vue SPA from fsys as the NoRoute fallback:
// real files (index.html, /assets/*) are served directly; any other GET path is
// a client-side route and gets index.html (history-mode fallback). Non-GET or
// unmatched API paths return a JSON 404.
func registerSPA(r *gin.Engine, fsys fs.FS) {
	fileServer := http.FileServer(http.FS(fsys))
	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if c.Request.Method != http.MethodGet || isAPIPath(p) {
			Error(c, http.StatusNotFound, "not found")
			return
		}
		name := strings.TrimPrefix(p, "/")
		if name == "" {
			name = "index.html"
		}
		if f, err := fsys.Open(name); err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}
		// Unknown path with no matching file → SPA client route → index.html.
		data, err := fs.ReadFile(fsys, "index.html")
		if err != nil {
			Error(c, http.StatusNotFound, "not found")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})
}
