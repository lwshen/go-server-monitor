package ws

import (
	"context"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/lwshen/go-server-monitor/internal/models"
)

const (
	sendBuffer   = 16               // per-client queue depth; overflow drops (REQ-WS-03)
	pingInterval = 15 * time.Second // server heartbeat cadence
	pongTimeout  = 30 * time.Second // no inbound frame within this -> connection dead
	writeTimeout = 10 * time.Second // per-frame write deadline
)

// Client represents one WebSocket connection (REQ-WS-05). Frames queue on send
// (buffered) so a slow consumer never blocks the Hub — overflow is dropped.
type Client struct {
	hub  *Hub
	conn *websocket.Conn

	subscribeScope string // "all" or a serverId
	send           chan models.WsMessage
}

// NewClient creates a client bound to hub with the given subscribe scope.
func NewClient(hub *Hub, conn *websocket.Conn, scope string) *Client {
	return &Client{
		hub:            hub,
		conn:           conn,
		subscribeScope: scope,
		send:           make(chan models.WsMessage, sendBuffer),
	}
}

// shouldReceive reports whether this client should receive a message tagged with
// the given scope (a serverId, or "all" for a global broadcast) — REQ-WS-05.
func (c *Client) shouldReceive(scope string) bool {
	if scope == "all" {
		return true
	}
	return c.subscribeScope == "all" || c.subscribeScope == scope
}

// ReadPump reads inbound frames until error/close, replying pong to an app-level
// ping. Any successful read is treated as liveness; a read exceeding pongTimeout
// (no pong / no traffic) tears the connection down. Runs until the connection
// closes, then unregisters.
func (c *Client) ReadPump(ctx context.Context) {
	defer c.close()
	for {
		rctx, cancel := context.WithTimeout(ctx, pongTimeout)
		var msg models.WsMessage
		err := wsjson.Read(rctx, c.conn, &msg)
		cancel()
		if err != nil {
			return
		}
		if msg.Type == "ping" {
			c.enqueue(models.WsMessage{Type: "pong"})
		}
	}
}

// WritePump drains send to the socket and emits a periodic ping. Exits when send
// is closed (hub unregistered the client), on write error, or on ctx cancel.
func (c *Client) WritePump(ctx context.Context) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	defer func() { _ = c.conn.Close(websocket.StatusNormalClosure, "closing") }()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			if err := c.write(ctx, msg); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.write(ctx, models.WsMessage{Type: "ping"}); err != nil {
				return
			}
		}
	}
}

// enqueue queues a frame without blocking; a full queue drops it (slow consumer).
func (c *Client) enqueue(msg models.WsMessage) {
	select {
	case c.send <- msg:
	default:
	}
}

func (c *Client) write(ctx context.Context, msg models.WsMessage) error {
	wctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()
	return wsjson.Write(wctx, c.conn, msg)
}

// close unregisters the client (idempotent in the Hub) and closes the socket.
func (c *Client) close() {
	c.hub.Unregister(c)
	_ = c.conn.Close(websocket.StatusNormalClosure, "closing")
}
