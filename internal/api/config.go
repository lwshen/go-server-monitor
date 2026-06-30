package api

import "github.com/gin-gonic/gin"

// Config returns public runtime config for the frontend (GET /api/config).
//
// P0 STUB: 501.
//
// TODO(P5/P6): return non-secret settings (site_title, theme_default,
// lang_default, is_public, captcha_provider/site_key, ...) from the settings
// table. Never include secret values.
func (h *Handlers) Config(c *gin.Context) {
	notImplemented(c)
}
