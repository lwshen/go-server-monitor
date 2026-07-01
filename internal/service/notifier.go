package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/lwshen/go-server-monitor/internal/store"
)

const (
	notifyMaxAttempts = 3
	notifyTimeout     = 10 * time.Second
)

// Alert is one notification to deliver (REQ-CRON-04). Type is offline/online/expiring.
type Alert struct {
	Type     string `json:"type"`
	ServerID string `json:"server_id"`
	Message  string `json:"message"`
}

// Notifier delivers alerts to the channels configured in the settings table
// (REQ-CRON-08). Settings are read at send time so admin changes take effect
// without a restart; when notify_enabled is not "true" (or no channel is
// configured) sends are a logged no-op.
type Notifier struct {
	st         store.Store
	log        *zap.Logger
	httpClient *http.Client
}

// NewNotifier constructs a Notifier backed by the settings store.
func NewNotifier(st store.Store, log *zap.Logger) *Notifier {
	return &Notifier{st: st, log: log, httpClient: &http.Client{Timeout: notifyTimeout}}
}

type notifyConfig struct {
	enabled bool
	tgToken string
	tgChat  string
	webhook string
}

func (n *Notifier) config(ctx context.Context) notifyConfig {
	get := func(k string) string {
		v, err := n.st.GetSetting(ctx, k)
		if err != nil {
			n.log.Warn("read setting failed", zap.String("key", k), zap.Error(err))
		}
		return v
	}
	return notifyConfig{
		enabled: get("notify_enabled") == "true",
		tgToken: get("telegram_bot_token"),
		tgChat:  get("telegram_chat_id"),
		webhook: get("webhook_url"),
	}
}

// Send delivers an alert to every configured channel. It returns nil when
// notifications are disabled or no channel is configured (a no-op), or the first
// delivery error otherwise; each channel is retried with exponential backoff.
func (n *Notifier) Send(ctx context.Context, a Alert) error {
	cfg := n.config(ctx)
	if !cfg.enabled {
		n.log.Debug("notify disabled, skipping", zap.String("type", a.Type), zap.String("server_id", a.ServerID))
		return nil
	}

	var firstErr error
	if cfg.tgToken != "" && cfg.tgChat != "" {
		if err := n.retry(ctx, func() error { return n.sendTelegram(ctx, cfg, a.Message) }); err != nil {
			n.log.Warn("telegram notify failed", zap.Error(err))
			firstErr = err
		}
	}
	if cfg.webhook != "" {
		if err := n.retry(ctx, func() error { return n.sendWebhook(ctx, cfg.webhook, a) }); err != nil {
			n.log.Warn("webhook notify failed", zap.Error(err))
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	if firstErr == nil {
		n.log.Info("alert sent", zap.String("type", a.Type), zap.String("server_id", a.ServerID))
	}
	return firstErr
}

// retry runs fn up to notifyMaxAttempts times with 1s/2s backoff.
func (n *Notifier) retry(ctx context.Context, fn func() error) error {
	var err error
	for attempt := 1; attempt <= notifyMaxAttempts; attempt++ {
		if err = fn(); err == nil {
			return nil
		}
		if attempt < notifyMaxAttempts {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(1<<(attempt-1)) * time.Second):
			}
		}
	}
	return err
}

func (n *Notifier) sendTelegram(ctx context.Context, cfg notifyConfig, text string) error {
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.tgToken)
	form := url.Values{"chat_id": {cfg.tgChat}, "text": {text}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return n.doExpectOK(req)
}

func (n *Notifier) sendWebhook(ctx context.Context, webhookURL string, a Alert) error {
	body, _ := json.Marshal(map[string]any{
		"type":      a.Type,
		"server_id": a.ServerID,
		"message":   a.Message,
		"timestamp": time.Now().Unix(),
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return n.doExpectOK(req)
}

func (n *Notifier) doExpectOK(req *http.Request) error {
	resp, err := n.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("notify endpoint returned HTTP %d", resp.StatusCode)
	}
	return nil
}
