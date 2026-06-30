// Package apperr defines a small application error type carrying an HTTP/business
// code alongside a human-readable message. It maps directly onto the frozen error
// response shape {"error":"<msg>","code":<n>} (CONVENTIONS.md §6).
package apperr

import "fmt"

// AppError is an error with an associated numeric code (usually an HTTP status).
type AppError struct {
	Code int    // HTTP status or business code
	Msg  string // human-readable message
}

// Error implements the error interface.
func (e *AppError) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Msg)
}

// New constructs an AppError with the given code and message.
func New(code int, msg string) *AppError {
	return &AppError{Code: code, Msg: msg}
}

// Newf constructs an AppError with a formatted message.
func Newf(code int, format string, args ...any) *AppError {
	return &AppError{Code: code, Msg: fmt.Sprintf(format, args...)}
}

// Common sentinel errors used across the codebase. Real handlers in P2+ may wrap
// these or construct fresh ones with more specific messages.
var (
	ErrUnauthorized   = New(401, "unauthorized")
	ErrNotFound       = New(404, "not found")
	ErrBadRequest     = New(400, "bad request")
	ErrInternal       = New(500, "internal server error")
	ErrNotImplemented = New(501, "not implemented")
)
