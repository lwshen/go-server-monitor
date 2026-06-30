package api

import "github.com/gin-gonic/gin"

// ServerDetail returns one server's detail (GET /api/server?id=<id>).
//
// P0 STUB: 501.
//
// TODO(P2): read ?id=, fetch the server's config + latest metrics + sys_info +
// ip_info; 404 if the id is unknown.
func (h *Handlers) ServerDetail(c *gin.Context) {
	notImplemented(c)
}
