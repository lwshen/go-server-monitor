package cron

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/lwshen/go-server-monitor/internal/service"
)

// expiryWindowDays is how far ahead an expiry triggers a reminder.
const expiryWindowDays = 7

// CheckExpirations sends a one-time reminder for servers whose expire_date is
// within expiryWindowDays (or already passed) and not yet notified (REQ-CRON-07).
func CheckExpirations(deps Deps) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	states, err := deps.Store.ListServerStates(ctx)
	if err != nil {
		deps.Log.Error("到期检查失败", zap.Error(err))
		return
	}

	for _, s := range states {
		if s.ExpireDate == "" || s.ExpirationNotified == 1 {
			continue
		}
		exp, err := time.Parse("2006-01-02", s.ExpireDate)
		if err != nil {
			deps.Log.Warn("expire_date 解析失败", zap.String("server", s.Name), zap.String("value", s.ExpireDate))
			continue
		}

		days := int(time.Until(exp).Hours() / 24)
		if days > expiryWindowDays {
			continue // not close enough yet
		}

		msg := fmt.Sprintf("[到期提醒] %s 将在 %d 天后到期（%s）", s.Name, days, s.ExpireDate)
		if days < 0 {
			msg = fmt.Sprintf("[已到期] %s 已于 %s 到期", s.Name, s.ExpireDate)
		}
		if err := deps.Notifier.Send(ctx, service.Alert{Type: "expiring", ServerID: s.ID, Message: msg}); err != nil {
			deps.Log.Warn("发送到期提醒失败", zap.String("server_id", s.ID), zap.Error(err))
		}
		if err := deps.Store.MarkExpirationNotified(ctx, s.ID); err != nil {
			deps.Log.Error("标记到期通知失败", zap.String("server_id", s.ID), zap.Error(err))
		}
		deps.Log.Info("到期提醒已发送", zap.String("server", s.Name), zap.Int("days", days))
	}
}
