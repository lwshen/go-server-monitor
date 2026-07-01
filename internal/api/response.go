// Package api wires the HTTP layer: the gin router, response helpers and one
// handler file per frozen endpoint. P0 registers every frozen route to a stub.
package api

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
)

// JSON writes obj as JSON with the given status code and the frozen content type
// (application/json; charset=utf-8 — gin's c.JSON sets this).
func JSON(c *gin.Context, code int, obj any) {
	c.JSON(code, obj)
}

// bindJSON unmarshals raw onto v. Unmarshalling onto a pre-populated struct yields
// partial-update semantics: only keys present in raw are overwritten.
func bindJSON(raw []byte, v any) error {
	return json.Unmarshal(raw, v)
}
