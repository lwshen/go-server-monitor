package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ServerDetail returns one server's detail (GET /api/server?id=<id>): config +
// display metadata + latest metrics + sys_info/ip_info snapshot. 404 if unknown.
func (h *Handlers) ServerDetail(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		Error(c, http.StatusBadRequest, "missing id")
		return
	}

	ctx := c.Request.Context()
	srv, err := h.deps.Store.GetServer(ctx, id)
	if err != nil {
		ErrorFrom(c, err)
		return
	}
	if srv == nil {
		Error(c, http.StatusNotFound, "server not found")
		return
	}

	srv.Online = isOnline(srv.LastUpdated, srv.ReportInterval, h.offlineFactor(ctx), time.Now().Unix())
	JSON(c, http.StatusOK, srv)
}
