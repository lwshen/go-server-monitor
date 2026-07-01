package cron

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/lwshen/go-server-monitor/internal/service"
)

// DetectOfflineServers flips servers between online/offline and alerts only on
// transitions (REQ-CRON-05): a server is offline when its newest sample is older
// than offline_factor × report_interval. Servers that have never reported are
// skipped (nothing to transition from).
func DetectOfflineServers(deps Deps) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	states, err := deps.Store.ListServerStates(ctx)
	if err != nil {
		deps.Log.Error("离线检测失败", zap.Error(err))
		return
	}

	factor := deps.Cfg.OfflineFactor
	if factor <= 0 {
		factor = 5
	}
	now := time.Now().Unix()

	for _, s := range states {
		if s.LastSeen == 0 {
			continue // never reported — don't alert on a freshly-registered server
		}
		reportInterval := s.ReportInterval
		if reportInterval <= 0 {
			reportInterval = 60
		}
		threshold := int64(reportInterval * factor)

		offline := now-s.LastSeen > threshold
		wasOnline := s.LastOnlineState == 1
		if offline == !wasOnline {
			continue // no state change
		}

		if err := deps.Store.SetOnlineState(ctx, s.ID, !offline, now); err != nil {
			deps.Log.Error("更新在线状态失败", zap.String("server_id", s.ID), zap.Error(err))
			continue
		}

		var alert service.Alert
		if offline {
			mins := (now - s.LastSeen) / 60
			deps.Log.Warn("离线检测：服务器离线", zap.String("server", s.Name), zap.Int64("minutes", mins))
			alert = service.Alert{Type: "offline", ServerID: s.ID,
				Message: fmt.Sprintf("[离线] %s（%s）已离线约 %d 分钟，请检查", s.Name, s.Location, mins)}
		} else {
			deps.Log.Info("离线检测：服务器恢复", zap.String("server", s.Name))
			alert = service.Alert{Type: "online", ServerID: s.ID,
				Message: fmt.Sprintf("[恢复] %s（%s）已恢复上报", s.Name, s.Location)}
		}
		if err := deps.Notifier.Send(ctx, alert); err != nil {
			deps.Log.Warn("发送告警失败", zap.String("server_id", s.ID), zap.Error(err))
		}
	}
}
