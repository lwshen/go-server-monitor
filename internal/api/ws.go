package api

import (
	"context"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/lwshen/go-server-monitor/internal/ws"
)

// WS upgrades to a WebSocket and subscribes the client (GET /ws?subscribe=all|<id>).
// It accepts the connection, registers a ws.Client with the Hub (which greets it
// with a "hello"), then runs the write pump in a goroutine and the read pump inline
// (blocking until the socket closes). Broadcasts arrive via the Hub on ingest.
func (h *Handlers) WS(c *gin.Context) {
	scope := c.DefaultQuery("subscribe", "all")

	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{
		// TODO(P8): restrict origins to CORS_ORIGINS. Behind Caddy the browser
		// origin differs from the upstream host, so verification is skipped here.
		InsecureSkipVerify: true,
	})
	if err != nil {
		h.deps.Log.Warn("ws accept failed", zap.Error(err))
		return
	}

	client := ws.NewClient(h.deps.Hub, conn, scope)
	h.deps.Hub.Register(client)

	ctx := context.Background()
	go client.WritePump(ctx)
	client.ReadPump(ctx) // blocks until the connection closes
}
