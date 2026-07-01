package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/lwshen/go-server-monitor/internal/models"
)

// AdminGetSettings reads global settings (GET /api/admin/settings, JWT). Secret
// keys are never returned in plaintext — each surfaces as a boolean "<key>_set"
// marker instead (REQ-RES-01).
func (h *Handlers) AdminGetSettings(c *gin.Context) {
	all, err := h.deps.Store.AllSettings(c.Request.Context())
	if err != nil {
		ErrorFrom(c, err)
		return
	}
	out := make(map[string]any, len(all))
	for k, v := range all {
		if models.SecretSettingKeys[k] {
			out[k+"_set"] = v != ""
			continue
		}
		out[k] = v
	}
	JSON(c, http.StatusOK, out)
}

// AdminPostSettings upserts UI-editable settings (POST /api/admin/settings, JWT).
// Write-protected/secret-bootstrap keys are silently skipped (REQ-RES-01).
func (h *Handlers) AdminPostSettings(c *gin.Context) {
	var body map[string]string
	if err := c.ShouldBindJSON(&body); err != nil {
		Error(c, http.StatusBadRequest, "invalid payload")
		return
	}

	ctx := c.Request.Context()
	updated := 0
	for k, v := range body {
		if models.WriteProtectedSettingKeys[k] {
			continue // e.g. api_secret / jwt_secret / admin_password_hash / schema_version
		}
		if err := h.deps.Store.SetSetting(ctx, k, v); err != nil {
			ErrorFrom(c, err)
			return
		}
		updated++
	}
	JSON(c, http.StatusOK, gin.H{"success": true, "updated": updated})
}
