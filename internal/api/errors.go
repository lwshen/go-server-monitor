package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/lwshen/go-server-monitor/pkg/apperr"
)

// errorBody is the frozen error response shape (CONVENTIONS.md §6):
//
//	{"error":"<msg>","code":<n>}
type errorBody struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}

// Error writes the frozen error response with the given HTTP status code and msg.
func Error(c *gin.Context, code int, msg string) {
	c.JSON(code, errorBody{Error: msg, Code: code})
}

// ErrorFrom writes an *apperr.AppError (or falls back to 500 for a plain error).
func ErrorFrom(c *gin.Context, err error) {
	if ae, ok := err.(*apperr.AppError); ok {
		Error(c, ae.Code, ae.Msg)
		return
	}
	Error(c, http.StatusInternalServerError, err.Error())
}

// notImplemented writes the standard 501 stub response used by every P0 handler
// that has no real logic yet.
func notImplemented(c *gin.Context) {
	Error(c, http.StatusNotImplemented, "not implemented")
}
