package api

import "github.com/gin-gonic/gin"

// Report ingests a probe upload (POST /report). The probe authenticates with a
// plaintext "secret" field in the JSON body, compared constant-time to api_secret
// (REQ-RES-05) — no HMAC.
//
// P0 STUB: 501.
//
// TODO(P2): bind models.ReportRequest, constant-time compare Secret against
// cfg.APISecret (crypto/subtle.ConstantTimeCompare) -> 401 on mismatch, then call
// service.SaveMetrics for each sample. Success returns 200 {"code":200,"message":"OK"}.
func (h *Handlers) Report(c *gin.Context) {
	notImplemented(c)
}
