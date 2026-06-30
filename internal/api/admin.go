package api

import "github.com/gin-gonic/gin"

// AdminLogin authenticates an admin and issues a JWT (POST /api/admin/login).
// Body is JSON {username,password}; on success returns a 7-day HS256 JWT
// (REQ-RES-05).
//
// P0 STUB: 501. service.CheckPassword / service.IssueJWT are real wrappers ready
// to use.
//
// TODO(P6): bind {username,password}; compare username to admin_username and
// bcrypt-verify against admin_password_hash; on success return
// {token, expires_in: 604800}.
func (h *Handlers) AdminLogin(c *gin.Context) {
	notImplemented(c)
}
