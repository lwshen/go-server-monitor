package api

import "github.com/gin-gonic/gin"

// AdminGetSettings reads global settings (GET /api/admin/settings, JWT).
//
// P0 STUB: 501.
//
// TODO(P6): return the UI-editable settings (REQ-RES-01). Secret fields
// (api_secret, jwt_secret, admin_password_hash, captcha_secret) MUST NOT be
// returned in plaintext — return a boolean "is set" marker instead (REQ-RES-01).
func (h *Handlers) AdminGetSettings(c *gin.Context) {
	notImplemented(c)
}

// AdminPostSettings writes global settings (POST /api/admin/settings, JWT).
//
// P0 STUB: 501.
//
// TODO(P6): upsert provided keys into the settings table; ignore attempts to set
// read-only secret-class keys via this endpoint.
func (h *Handlers) AdminPostSettings(c *gin.Context) {
	notImplemented(c)
}
