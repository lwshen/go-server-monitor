package probe

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lwshen/go-server-monitor/internal/models"
)

func testConfig(url string) *Config {
	return &Config{
		ServerURL:       url,
		ServerID:        "srv-x",
		APISecret:       "sek",
		CollectInterval: 5 * time.Second,
		ReportInterval:  60 * time.Second,
	}
}

// TestUploadEnvelope verifies a batch upload sends the 03-report-protocol envelope:
// id + secret in the body, one Samples entry per buffered report, Data = newest.
func TestUploadEnvelope(t *testing.T) {
	var hits int32
	var got models.ReportRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":200,"message":"OK"}`))
	}))
	defer srv.Close()

	u := NewUploader(testConfig(srv.URL))
	samples := []*models.StatReport{
		{Cpu: 1, LatestTs: 100},
		{Cpu: 2, LatestTs: 200},
	}
	if err := u.Upload(samples); err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if hits != 1 {
		t.Fatalf("server hits = %d, want 1", hits)
	}
	if got.ID != "srv-x" || got.Secret != "sek" {
		t.Fatalf("envelope id/secret = %q/%q, want srv-x/sek", got.ID, got.Secret)
	}
	if len(got.Samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(got.Samples))
	}
	if got.Data == nil || got.Data.Cpu != 2 {
		t.Fatalf("Data = %+v, want newest sample (cpu 2)", got.Data)
	}
	if got.CollectInterval != 5 || got.ReportInterval != 60 {
		t.Fatalf("intervals = %d/%d, want 5/60", got.CollectInterval, got.ReportInterval)
	}
}

// TestUploadSingleSampleNoBatch: a lone sample goes in Data with no Samples array.
func TestUploadSingleSampleNoBatch(t *testing.T) {
	var got models.ReportRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if err := NewUploader(testConfig(srv.URL)).Upload([]*models.StatReport{{Cpu: 7, LatestTs: 1}}); err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if len(got.Samples) != 0 || got.Data == nil || got.Data.Cpu != 7 {
		t.Fatalf("single sample = Data %+v / Samples %d, want Data cpu 7 / 0 samples", got.Data, len(got.Samples))
	}
}

// TestUploadNoRetryOn4xx: a client error (e.g. bad secret / unknown id) is permanent.
func TestUploadNoRetryOn4xx(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	err := NewUploader(testConfig(srv.URL)).Upload([]*models.StatReport{{LatestTs: 1}})
	if err == nil {
		t.Fatal("Upload should fail on 401")
	}
	if hits != 1 {
		t.Fatalf("hits = %d, want 1 (no retry on 4xx)", hits)
	}
}

// TestUploadRetryOn5xx: server errors are retried up to maxUploadAttempts.
func TestUploadRetryOn5xx(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	err := NewUploader(testConfig(srv.URL)).Upload([]*models.StatReport{{LatestTs: 1}})
	if err == nil {
		t.Fatal("Upload should fail after retries")
	}
	if int(hits) != maxUploadAttempts {
		t.Fatalf("hits = %d, want %d (retried)", hits, maxUploadAttempts)
	}
}
