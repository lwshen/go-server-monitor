package probe

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/lwshen/go-server-monitor/internal/models"
)

// maxUploadAttempts bounds the retry loop; delays back off 1s, 2s, 4s.
const maxUploadAttempts = 3

// Uploader POSTs collected samples to the server's /report endpoint (REQ-PROBE-09).
type Uploader struct {
	cfg        *Config
	endpoint   string
	httpClient *http.Client
}

// NewUploader creates an Uploader targeting cfg.ServerURL + "/report".
func NewUploader(cfg *Config) *Uploader {
	return &Uploader{
		cfg:        cfg,
		endpoint:   strings.TrimRight(cfg.ServerURL, "/") + "/report",
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Upload sends the samples as one /report envelope (REQ-PROBE-09 / 03-report-protocol
// §2.2): the newest sample is Data, all samples go in Samples (server writes one
// metrics_history row each, REQ-RES-04), secret travels in the body (REQ-RES-05).
//
// 2xx = success. 4xx is a permanent client error (bad secret, unregistered id) and
// is NOT retried — the caller should stop and fix config. 5xx / network errors are
// retried with exponential backoff.
func (u *Uploader) Upload(samples []*models.StatReport) error {
	if len(samples) == 0 {
		return nil
	}
	latest := samples[len(samples)-1] // buffer preserves collection order

	req := models.ReportRequest{
		ID:              u.cfg.ServerID,
		Secret:          u.cfg.APISecret,
		Timestamp:       time.Now().Unix(),
		Data:            latest,
		CollectInterval: int(u.cfg.CollectInterval.Seconds()),
		ReportInterval:  int(u.cfg.ReportInterval.Seconds()),
	}
	if len(samples) > 1 {
		req.Samples = make([]models.ReportSample, len(samples))
		for i, s := range samples {
			req.Samples[i] = models.ReportSample{Timestamp: s.LatestTs, Data: s}
		}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}

	var lastErr error
	for attempt := 1; attempt <= maxUploadAttempts; attempt++ {
		status, respBody, err := u.post(body)
		switch {
		case err != nil:
			lastErr = err
		case status >= 200 && status < 300:
			return nil
		case status >= 400 && status < 500:
			// Permanent: no point retrying a rejected secret / unknown id.
			return fmt.Errorf("report rejected: HTTP %d: %s", status, strings.TrimSpace(respBody))
		default:
			lastErr = fmt.Errorf("report failed: HTTP %d: %s", status, strings.TrimSpace(respBody))
		}
		if attempt < maxUploadAttempts {
			time.Sleep(time.Duration(1<<(attempt-1)) * time.Second) // 1s, 2s
		}
	}
	return fmt.Errorf("upload failed after %d attempts: %w", maxUploadAttempts, lastErr)
}

func (u *Uploader) post(body []byte) (status int, respBody string, err error) {
	resp, err := u.httpClient.Post(u.endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return resp.StatusCode, string(b), nil
}
