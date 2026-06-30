// Package cron registers the scheduled jobs: offline detection, retention cleanup
// and expiration reminders (REQ-CRON-09). P0 registers all three as logging stubs.
package cron

import (
	"github.com/robfig/cron/v3"

	"github.com/lwshen/go-server-monitor/internal/config"
	"github.com/lwshen/go-server-monitor/internal/service"
	"github.com/lwshen/go-server-monitor/internal/store"
	"go.uber.org/zap"
)

// Deps bundles what the cron jobs need to run.
type Deps struct {
	Store    store.Store
	Cfg      *config.Config
	Notifier *service.Notifier
	Log      *zap.Logger
}

// Start creates a cron scheduler, registers the three jobs and starts it.
// The returned *cron.Cron is owned by the caller (main.go) for graceful Stop().
//
// Schedules (REQ-CRON-10): offline every */5 min, cleanup daily at UTC 00:00,
// expiration reminder hourly. The job bodies are P7 stubs that only log.
func Start(deps Deps) (*cron.Cron, error) {
	c := cron.New()

	if _, err := c.AddFunc("*/5 * * * *", func() { DetectOfflineServers(deps) }); err != nil {
		return nil, err
	}
	if _, err := c.AddFunc("0 0 * * *", func() { CleanupOldMetrics(deps) }); err != nil {
		return nil, err
	}
	if _, err := c.AddFunc("0 * * * *", func() { CheckExpirations(deps) }); err != nil {
		return nil, err
	}

	c.Start()
	deps.Log.Info("Cron 启动成功", zap.Int("jobs", len(c.Entries())))
	return c, nil
}
