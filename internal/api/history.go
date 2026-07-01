package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// History returns server-side downsampled history (GET /api/history?id=<id>&range=<r>).
// range is one of 1h/6h/24h/7d/30d/180d (default 1h); the store buckets and
// averages per REQ-RES-03 (ping_*/loss_* ignore NULLs).
func (h *Handlers) History(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		Error(c, http.StatusBadRequest, "missing id")
		return
	}
	rng := c.DefaultQuery("range", "1h")

	points, err := h.deps.Store.QueryHistory(c.Request.Context(), id, rng)
	if err != nil {
		ErrorFrom(c, err) // invalid range surfaces as a 400 apperr
		return
	}

	JSON(c, http.StatusOK, gin.H{
		"id":      id,
		"range":   rng,
		"samples": points,
	})
}
