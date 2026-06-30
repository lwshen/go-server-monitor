// Package middleware provides gin middleware: JWT auth, CORS and request logging.
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/lwshen/go-server-monitor/internal/service"
)

// JWTAuth guards admin routes by validating the Bearer JWT (REQ-ADMIN-04 /
// REQ-SEC-02). It expects "Authorization: Bearer <jwt>".
//
// P0 STUB: the wiring (extract header, parse token, set claims, abort 401) is
// present and compile-clean, but it is intentionally permissive-by-config in the
// skeleton: with an empty jwtSecret it lets requests through so the admin stubs
// can be exercised. With a non-empty secret it performs a real HS256 validation.
//
// TODO(P6): always require a valid token; surface claims via c.Set and drop the
// empty-secret passthrough.
func JWTAuth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skeleton passthrough: no secret configured yet.
		if jwtSecret == "" {
			c.Next()
			return
		}

		authz := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(authz, prefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing bearer token",
				"code":  http.StatusUnauthorized,
			})
			return
		}

		token := strings.TrimPrefix(authz, prefix)
		claims, err := service.ParseJWT(jwtSecret, token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired token",
				"code":  http.StatusUnauthorized,
			})
			return
		}

		c.Set("claims", claims)
		c.Next()
	}
}
