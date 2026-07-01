package store

import (
	"context"
	"errors"
	"testing"

	"github.com/lwshen/go-server-monitor/internal/models"
	"github.com/lwshen/go-server-monitor/pkg/apperr"
)

// mustCreateServer registers a server so /report (which 404s on unknown ids) can
// accept reports for it.
func mustCreateServer(t *testing.T, st Store, id, name string) {
	t.Helper()
	if err := st.CreateServer(context.Background(), &models.ServerConfig{ID: id, Name: name}); err != nil {
		t.Fatalf("CreateServer(%s): %v", id, err)
	}
}

func sampleReport(name string, cpu float64, ts int64) *models.StatReport {
	return &models.StatReport{
		Name: name, Gid: "g1", Alias: "主力", Type: "cloud", Location: "US-East",
		Notify: true, Weight: 3, LatestTs: ts,
		Cpu: cpu, MemoryTotal: 8192, MemoryUsed: 2048,
		Load1: 1.5, Load5: 1.2, Load15: 0.9,
		PingCt: 32, PingCu: -1, // cu unmeasured -> NULL
		LossCt: 0, LossCu: -1, // ct 0% is real, cu unmeasured -> NULL
		Online4: true,
		Disks:   []models.DiskInfo{{Name: "sda1", MountPoint: "/", FileSystem: "ext4", Total: 100, Used: 40, Free: 60}},
		SysInfo: &models.SysInfo{OsName: "Linux", OsArch: "x86_64", CpuNum: 4, CpuBrand: "Xeon", HostName: "h1"},
		IpInfo:  &models.IpInfo{Country: "US", Lat: 39.0, Lon: -77.4},
	}
}

func TestSaveReportSingle(t *testing.T) {
	ctx := context.Background()
	st, _ := openTemp(t)
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	mustCreateServer(t, st, "srv-1", "prod-01")

	req := &models.ReportRequest{
		ID: "srv-1", Secret: "x", Timestamp: 1_700_000_000,
		Data: sampleReport("prod-01", 25.5, 1_700_000_000),
	}
	n, err := st.SaveReport(ctx, req)
	if err != nil || n != 1 {
		t.Fatalf("SaveReport = (%d, %v), want (1, nil)", n, err)
	}

	bs := st.(*bunStore)
	var rows int
	if err := bs.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM metrics_history WHERE server_id='srv-1'`).Scan(&rows); err != nil {
		t.Fatalf("count: %v", err)
	}
	if rows != 1 {
		t.Fatalf("metrics rows = %d, want 1", rows)
	}

	// -1 sentinels stored as NULL; real values stored.
	var pingCt, lossCt *float64
	var pingCu, lossCu *float64
	if err := bs.db.QueryRowContext(ctx,
		`SELECT ping_ct, loss_ct, ping_cu, loss_cu FROM metrics_history WHERE server_id='srv-1'`).
		Scan(&pingCt, &lossCt, &pingCu, &lossCu); err != nil {
		t.Fatalf("scan quality: %v", err)
	}
	if pingCt == nil || *pingCt != 32 {
		t.Fatalf("ping_ct = %v, want 32", pingCt)
	}
	if lossCt == nil || *lossCt != 0 {
		t.Fatalf("loss_ct = %v, want 0 (real)", lossCt)
	}
	if pingCu != nil || lossCu != nil {
		t.Fatalf("ping_cu/loss_cu = %v/%v, want NULL (unmeasured -1)", pingCu, lossCu)
	}

	// admin name preserved; probe metadata + ip_info snapshot refreshed.
	var name, alias, region, ipJSON, loadAvg string
	if err := bs.db.QueryRowContext(ctx,
		`SELECT s.name, s.alias, m.region, s.ip_info_json, m.load_avg
		   FROM servers s JOIN metrics_history m ON m.server_id=s.id WHERE s.id='srv-1'`).
		Scan(&name, &alias, &region, &ipJSON, &loadAvg); err != nil {
		t.Fatalf("scan server: %v", err)
	}
	if name != "prod-01" || alias != "主力" {
		t.Fatalf("server = (%q,%q), want (prod-01,主力)", name, alias)
	}
	if region != "US" {
		t.Fatalf("region = %q, want US (injected from ip_info)", region)
	}
	if ipJSON == "" || ipJSON == "null" {
		t.Fatalf("ip_info_json = %q, want JSON snapshot", ipJSON)
	}
	if loadAvg != "1.5 1.2 0.9" {
		t.Fatalf("load_avg = %q, want \"1.5 1.2 0.9\"", loadAvg)
	}
}

// TestSaveReportBatch verifies a batched upload writes one row per sample
// (REQ-RES-04) and re-reporting the same id updates (not duplicates) the server.
func TestSaveReportBatch(t *testing.T) {
	ctx := context.Background()
	st, _ := openTemp(t)
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	mustCreateServer(t, st, "srv-2", "batch-host")

	req := &models.ReportRequest{
		ID: "srv-2", Secret: "x", CollectInterval: 5, ReportInterval: 15,
		Samples: []models.ReportSample{
			{Timestamp: 1_700_000_000, Data: sampleReport("h", 10, 1_700_000_000)},
			{Timestamp: 1_700_000_005, Data: sampleReport("h", 20, 1_700_000_005)},
			{Timestamp: 1_700_000_010, Data: sampleReport("h", 30, 1_700_000_010)},
		},
	}
	n, err := st.SaveReport(ctx, req)
	if err != nil || n != 3 {
		t.Fatalf("SaveReport batch = (%d, %v), want (3, nil)", n, err)
	}
	// report again -> 3 more metric rows, still ONE server row.
	if _, err := st.SaveReport(ctx, req); err != nil {
		t.Fatalf("SaveReport second: %v", err)
	}

	bs := st.(*bunStore)
	var metrics, servers int
	bs.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM metrics_history WHERE server_id='srv-2'`).Scan(&metrics)
	bs.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM servers WHERE id='srv-2'`).Scan(&servers)
	if metrics != 6 {
		t.Fatalf("metrics rows = %d, want 6", metrics)
	}
	if servers != 1 {
		t.Fatalf("server rows = %d, want 1 (update, not duplicate)", servers)
	}
}

// TestSaveReportUnknownServer verifies an unregistered id is rejected with 404 and
// writes nothing (CONVENTIONS §6 / 03-report-protocol §2.7.2).
func TestSaveReportUnknownServer(t *testing.T) {
	ctx := context.Background()
	st, _ := openTemp(t)
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	_, err := st.SaveReport(ctx, &models.ReportRequest{
		ID: "ghost", Data: sampleReport("x", 1, 1_700_000_000),
	})
	var ae *apperr.AppError
	if !errors.As(err, &ae) || ae.Code != 404 {
		t.Fatalf("SaveReport(unknown id) err = %v, want AppError code 404", err)
	}

	bs := st.(*bunStore)
	var rows int
	bs.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM metrics_history WHERE server_id='ghost'`).Scan(&rows)
	if rows != 0 {
		t.Fatalf("metrics rows for ghost = %d, want 0 (nothing written)", rows)
	}
}
