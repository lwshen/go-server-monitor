package probe

import (
	"net/http"
	"time"

	"github.com/lwshen/go-server-monitor/internal/models"
	"github.com/lwshen/go-server-monitor/pkg/apperr"
)

// Uploader POSTs collected samples to the server's /report endpoint (REQ-PROBE-09).
type Uploader struct {
	serverURL  string
	apiSecret  string
	httpClient *http.Client
}

// NewUploader creates an Uploader with a 30s HTTP timeout.
func NewUploader(serverURL, apiSecret string) *Uploader {
	return &Uploader{
		serverURL:  serverURL,
		apiSecret:  apiSecret,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Upload sends the given samples to the server (REQ-PROBE-09).
//
// P0 STUB: returns ErrNotImplemented.
//
// TODO(P3): build a models.ReportRequest (secret in body, REQ-RES-05), JSON-encode
// it, POST to serverURL+"/report" with retry (3x), and treat 200 as success.
func (u *Uploader) Upload(samples []*models.StatReport) error {
	_ = samples
	return apperr.ErrNotImplemented
}
