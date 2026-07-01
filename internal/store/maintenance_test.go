package store

import (
	"context"
	"testing"
	"time"

	"github.com/lwshen/go-server-monitor/internal/models"
)

func TestDeleteMetricsBefore(t *testing.T) {
	ctx := context.Background()
	st, _ := openTemp(t)
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	mustCreateServer(t, st, "s1", "n1")
	now := time.Now().Unix()

	report := func(ts int64) {
		if _, err := st.SaveReport(ctx, &models.ReportRequest{
			ID: "s1", Data: &models.StatReport{LatestTs: ts, Cpu: 1},
		}); err != nil {
			t.Fatalf("SaveReport(%d): %v", ts, err)
		}
	}
	report(now - 1_000_000) // old
	report(now)             // fresh

	deleted, err := st.DeleteMetricsBefore(ctx, now-500_000)
	if err != nil || deleted != 1 {
		t.Fatalf("DeleteMetricsBefore = (%d, %v), want (1, nil)", deleted, err)
	}

	bs := st.(*bunStore)
	var remaining int
	bs.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM metrics_history WHERE server_id='s1'`).Scan(&remaining)
	if remaining != 1 {
		t.Fatalf("remaining rows = %d, want 1", remaining)
	}
}

func TestListServerStates(t *testing.T) {
	ctx := context.Background()
	st, _ := openTemp(t)
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	mustCreateServer(t, st, "s1", "reported")
	mustCreateServer(t, st, "s2", "silent")
	now := time.Now().Unix()
	if _, err := st.SaveReport(ctx, &models.ReportRequest{ID: "s1", Data: &models.StatReport{LatestTs: now, Cpu: 1}}); err != nil {
		t.Fatalf("SaveReport: %v", err)
	}

	states, err := st.ListServerStates(ctx)
	if err != nil {
		t.Fatalf("ListServerStates: %v", err)
	}
	byID := map[string]models.ServerState{}
	for _, s := range states {
		byID[s.ID] = s
	}
	if got := byID["s1"]; got.LastSeen != now || got.LastOnlineState != 1 {
		t.Fatalf("s1 = %+v, want last_seen=%d state=1", got, now)
	}
	if got := byID["s2"]; got.LastSeen != 0 {
		t.Fatalf("s2 last_seen = %d, want 0 (never reported)", got.LastSeen)
	}

	// SetOnlineState round-trip.
	if err := st.SetOnlineState(ctx, "s1", false, now); err != nil {
		t.Fatalf("SetOnlineState: %v", err)
	}
	states, _ = st.ListServerStates(ctx)
	for _, s := range states {
		if s.ID == "s1" && s.LastOnlineState != 0 {
			t.Fatalf("s1 state after SetOnlineState(false) = %d, want 0", s.LastOnlineState)
		}
	}
}
