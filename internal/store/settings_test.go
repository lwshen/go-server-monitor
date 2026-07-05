package store

import (
	"context"
	"testing"
)

func TestIntSetting(t *testing.T) {
	ctx := context.Background()
	st, _ := openTemp(t)
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// absent -> default
	if got := IntSetting(ctx, st, "offline_factor", 5); got != 5 {
		t.Fatalf("absent = %d, want default 5", got)
	}
	// present + valid -> parsed
	_ = st.SetSetting(ctx, "offline_factor", "3")
	if got := IntSetting(ctx, st, "offline_factor", 5); got != 3 {
		t.Fatalf("set 3 = %d, want 3", got)
	}
	// present but invalid -> default
	_ = st.SetSetting(ctx, "offline_factor", "not-an-int")
	if got := IntSetting(ctx, st, "offline_factor", 5); got != 5 {
		t.Fatalf("invalid = %d, want default 5", got)
	}
}
