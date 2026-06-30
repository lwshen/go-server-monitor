package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Health is the public health check (GET /health). Returns 200 {"status":"ok"}.
func (h *Handlers) Health(c *gin.Context) {
	JSON(c, http.StatusOK, gin.H{"status": "ok"})
}
