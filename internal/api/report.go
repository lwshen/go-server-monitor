package api

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/lwshen/go-server-monitor/internal/models"
	"github.com/lwshen/go-server-monitor/internal/service"
)

// Report ingests a probe upload (POST /report). The probe authenticates with a
// plaintext "secret" field in the JSON body, compared constant-time to
// api_secret (REQ-RES-05) — no HMAC; HTTPS provides confidentiality.
//
// Flow (REQ-API-05): reject if uploads are disabled (empty API_SECRET) → bind the
// envelope → constant-time secret check → require id + a payload → persist via
// service.Ingest (which writes one metrics_history row per sample and upserts the
// server). Success returns {"code":200,"message":"OK"}.
func (h *Handlers) Report(c *gin.Context) {
	secret := h.deps.Cfg.APISecret
	if secret == "" {
		Error(c, http.StatusUnauthorized, "uploads disabled: API_SECRET not configured")
		return
	}

	var req models.ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	// Constant-time comparison to avoid leaking the secret via timing.
	if subtle.ConstantTimeCompare([]byte(req.Secret), []byte(secret)) != 1 {
		Error(c, http.StatusUnauthorized, "invalid secret")
		return
	}

	if req.ID == "" {
		Error(c, http.StatusBadRequest, "missing server id")
		return
	}
	if req.Data == nil && len(req.Samples) == 0 {
		Error(c, http.StatusBadRequest, "missing metrics")
		return
	}

	n, err := service.Ingest(c.Request.Context(), h.deps.Store, h.deps.Hub, &req, h.deps.Log)
	if err != nil {
		ErrorFrom(c, err)
		return
	}

	JSON(c, http.StatusOK, gin.H{"code": 200, "message": "OK", "saved": n})
}
