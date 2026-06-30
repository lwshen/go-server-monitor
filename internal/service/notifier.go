package service

import (
	"github.com/lwshen/go-server-monitor/pkg/apperr"
	"go.uber.org/zap"
)

// Notifier sends alerts via Telegram and/or a generic webhook (REQ-CRON-08).
type Notifier struct {
	tgBotToken string
	tgChatID   string
	webhookURL string
	log        *zap.Logger
}

// NewNotifier constructs a Notifier from settings values.
//
// TODO(P7): source tokens/URL from the settings table at call time so admin
// changes take effect without a restart.
func NewNotifier(tgBotToken, tgChatID, webhookURL string, log *zap.Logger) *Notifier {
	return &Notifier{
		tgBotToken: tgBotToken,
		tgChatID:   tgChatID,
		webhookURL: webhookURL,
		log:        log,
	}
}

// Send delivers a message to all configured channels.
//
// P0 STUB: logs "not implemented" and returns ErrNotImplemented.
//
// TODO(P7): send Telegram + webhook concurrently with exponential-backoff retry
// (1s/2s/4s, max 3) and UUID-based idempotency.
func (n *Notifier) Send(msg string) error {
	n.log.Warn("service.Notifier.Send not implemented (P7)", zap.String("msg", msg))
	return apperr.ErrNotImplemented
}

// NotifyOffline composes and sends an offline alert for a server.
//
// P0 STUB: delegates to Send (which is itself a stub).
//
// TODO(P7): format the "[离线] <name> (<region>) ..." message and dedupe.
func (n *Notifier) NotifyOffline(serverID, serverName string) error {
	n.log.Warn("service.Notifier.NotifyOffline not implemented (P7)",
		zap.String("server_id", serverID), zap.String("server_name", serverName))
	return apperr.ErrNotImplemented
}
