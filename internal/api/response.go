// Package api wires the HTTP layer: the gin router, response helpers and one
// handler file per frozen endpoint. P0 registers every frozen route to a stub.
package api

import "github.com/gin-gonic/gin"

// JSON writes obj as JSON with the given status code and the frozen content type
// (application/json; charset=utf-8 — gin's c.JSON sets this).
func JSON(c *gin.Context, code int, obj any) {
	c.JSON(code, obj)
}
