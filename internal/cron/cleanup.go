package cron

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// CleanupOldMetrics deletes metrics_history rows older than the retention window
// (REQ-CRON-06 / REQ-DB-06). Runs nightly.
func CleanupOldMetrics(deps Deps) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	retention := deps.Cfg.ReportRetentionDays
	if retention <= 0 {
		retention = 180
	}
	cutoff := time.Now().Unix() - int64(retention)*86400

	deleted, err := deps.Store.DeleteMetricsBefore(ctx, cutoff)
	if err != nil {
		deps.Log.Error("数据清理失败", zap.Error(err))
		return
	}
	deps.Log.Info("数据清理完成",
		zap.Int64("deleted", deleted),
		zap.Int("retention_days", retention))
}
