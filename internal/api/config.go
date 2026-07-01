package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// publicConfigKeys are the non-secret settings exposed to the frontend. Secret
// keys (api_secret, jwt_secret, admin_password_hash, captcha_secret, …) are never
// included (REQ-RES-01).
var publicConfigKeys = []string{
	"site_title", "theme_default", "lang_default", "is_public",
	"captcha_provider", "captcha_site_key",
}

// Config returns public runtime config for the frontend (GET /api/config):
// built-in defaults overlaid with any non-secret values from the settings table.
func (h *Handlers) Config(c *gin.Context) {
	// Values are strings to match the frozen config contract (04-backend-api
	// REQ-API-02): the frontend auth gate compares is_public === "true".
	cfg := gin.H{
		"site_title":    "Server Monitor",
		"theme_default": "auto",
		"lang_default":  "zh",
		"is_public":     "true",
	}

	if all, err := h.deps.Store.AllSettings(c.Request.Context()); err == nil {
		for _, k := range publicConfigKeys {
			if v, ok := all[k]; ok && v != "" {
				cfg[k] = v
			}
		}
	}

	JSON(c, http.StatusOK, cfg)
}
