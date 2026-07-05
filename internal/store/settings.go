package store

import (
	"context"
	"strconv"
	"strings"
)

// IntSetting reads an integer setting, returning def when the key is absent,
// unreadable, or not a valid integer. Callers use it to make a DB setting the
// authoritative source for an operational value while keeping an env/config
// fallback (e.g. offline_factor, retention_days).
func IntSetting(ctx context.Context, st Store, key string, def int) int {
	v, err := st.GetSetting(ctx, key)
	if err != nil || v == "" {
		return def
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return def
	}
	return n
}
