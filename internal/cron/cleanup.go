package cron

import "go.uber.org/zap"

// CleanupOldMetrics deletes metrics_history rows older than the retention window
// (REQ-CRON-06 / REQ-DB-06).
//
// P0 STUB: logs "not implemented".
//
// TODO(P7): DELETE FROM metrics_history WHERE timestamp < now - retention_days
// (timestamp in Unix seconds), log rows deleted, then PRAGMA wal_checkpoint /
// optional VACUUM.
func CleanupOldMetrics(deps Deps) {
	deps.Log.Warn("cron.CleanupOldMetrics not implemented (P7)",
		zap.Int("retention_days", deps.Cfg.ReportRetentionDays))
}
