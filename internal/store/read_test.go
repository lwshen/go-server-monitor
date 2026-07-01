package store

import (
	"context"
	"testing"
	"time"

	"github.com/lwshen/go-server-monitor/internal/models"
)

func TestReadAPIs(t *testing.T) {
	ctx := context.Background()
	st, _ := openTemp(t)
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	now := time.Now().Unix()
	mustCreateServer(t, st, "srv-a", "alpha")
	mustCreateServer(t, st, "srv-b", "beta")

	// Server A: a single recent sample.
	if _, err := st.SaveReport(ctx, &models.ReportRequest{
		ID: "srv-a", Timestamp: now, Data: sampleReport("alpha", 42, now),
	}); err != nil {
		t.Fatalf("save A: %v", err)
	}
	// Server B: a batch of three recent samples.
	if _, err := st.SaveReport(ctx, &models.ReportRequest{
		ID: "srv-b",
		Samples: []models.ReportSample{
			{Timestamp: now - 10, Data: sampleReport("beta", 10, now-10)},
			{Timestamp: now - 5, Data: sampleReport("beta", 20, now-5)},
			{Timestamp: now, Data: sampleReport("beta", 30, now)},
		},
	}); err != nil {
		t.Fatalf("save B: %v", err)
	}

	// ── ListServers ──────────────────────────────────────────────────────────
	servers, err := st.ListServers(ctx)
	if err != nil {
		t.Fatalf("ListServers: %v", err)
	}
	if len(servers) != 2 {
		t.Fatalf("ListServers len = %d, want 2", len(servers))
	}
	byID := map[string]models.Server{}
	for _, s := range servers {
		byID[s.ID] = s
	}
	a := byID["srv-a"]
	if a.LatestMetrics == nil || a.LatestMetrics.Cpu != 42 || a.LastUpdated != now {
		t.Fatalf("srv-a latest = %+v / lastUpdated %d, want cpu 42 @ %d", a.LatestMetrics, a.LastUpdated, now)
	}
	if a.Alias != "主力" {
		t.Fatalf("srv-a alias = %q, want 主力", a.Alias)
	}

	// ── GetServer ────────────────────────────────────────────────────────────
	got, err := st.GetServer(ctx, "srv-a")
	if err != nil || got == nil {
		t.Fatalf("GetServer(srv-a) = (%v, %v)", got, err)
	}
	if got.SysInfo == nil || got.SysInfo.HostName != "h1" {
		t.Fatalf("GetServer sys_info = %+v, want HostName h1", got.SysInfo)
	}
	if got.IpInfo == nil || got.IpInfo.Country != "US" {
		t.Fatalf("GetServer ip_info = %+v, want Country US", got.IpInfo)
	}
	if missing, err := st.GetServer(ctx, "nope"); err != nil || missing != nil {
		t.Fatalf("GetServer(nope) = (%v, %v), want (nil, nil)", missing, err)
	}

	// ── QueryHistory ─────────────────────────────────────────────────────────
	pts, err := st.QueryHistory(ctx, "srv-b", "1h")
	if err != nil {
		t.Fatalf("QueryHistory: %v", err)
	}
	if len(pts) < 1 || len(pts) > 3 {
		t.Fatalf("QueryHistory points = %d, want 1..3", len(pts))
	}
	for _, p := range pts {
		if p.Cpu < 10 || p.Cpu > 30 {
			t.Fatalf("bucket cpu avg = %v, want within [10,30]", p.Cpu)
		}
		if p.PingCt == nil || *p.PingCt != 32 {
			t.Fatalf("bucket ping_ct = %v, want 32 (all samples 32ms)", p.PingCt)
		}
		if p.PingCu != nil {
			t.Fatalf("bucket ping_cu = %v, want nil (all unmeasured -1 -> NULL ignored by AVG)", *p.PingCu)
		}
		if p.LossCt == nil || *p.LossCt != 0 {
			t.Fatalf("bucket loss_ct = %v, want 0 (real)", p.LossCt)
		}
	}

	if _, err := st.QueryHistory(ctx, "srv-b", "bogus"); err == nil {
		t.Fatalf("QueryHistory(bogus range) should error")
	}
}
