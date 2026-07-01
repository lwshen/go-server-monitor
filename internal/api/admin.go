package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/lwshen/go-server-monitor/internal/models"
	"github.com/lwshen/go-server-monitor/internal/service"
)

// jwtExpiresIn is the admin token lifetime in seconds (7 days), echoed to clients.
const jwtExpiresIn = 7 * 24 * 60 * 60

// AdminLogin authenticates an admin and issues a JWT (POST /api/admin/login).
// Body is JSON {username,password}; on success returns a 7-day HS256 JWT
// (REQ-RES-05). Credentials come from the settings table (bootstrapped from .env).
func (h *Handlers) AdminLogin(c *gin.Context) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		Error(c, http.StatusBadRequest, "invalid payload")
		return
	}

	ctx := c.Request.Context()
	hash, err := h.deps.Store.GetSetting(ctx, models.SettingAdminPasswordHash)
	if err != nil {
		ErrorFrom(c, err)
		return
	}
	if hash == "" {
		Error(c, http.StatusUnauthorized, "admin login not configured (set ADMIN_PASSWORD)")
		return
	}
	username, _ := h.deps.Store.GetSetting(ctx, models.SettingAdminUsername)
	if username == "" {
		username = "admin"
	}

	// An omitted username defaults to the configured one (single-admin system).
	reqUser := body.Username
	if reqUser == "" {
		reqUser = username
	}
	if reqUser != username || !service.CheckPassword(hash, body.Password) {
		Error(c, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := service.IssueJWT(h.deps.Cfg.JWTSecret, "admin")
	if err != nil {
		ErrorFrom(c, err)
		return
	}
	JSON(c, http.StatusOK, gin.H{"token": token, "expires_in": jwtExpiresIn})
}
