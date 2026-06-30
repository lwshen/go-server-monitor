package ws

import (
	"time"

	"github.com/coder/websocket"
)

// Client represents one WebSocket connection (REQ-WS-01.2).
type Client struct {
	hub  *Hub
	conn *websocket.Conn // coder/websocket connection

	subscribeScope string           // "all" or a serverId
	send           chan interface{} // small buffer (10) to avoid head-of-line blocking

	lastHeartbeat time.Time
}

// NewClient creates a client bound to hub with the given subscribe scope.
func NewClient(hub *Hub, conn *websocket.Conn, scope string) *Client {
	return &Client{
		hub:            hub,
		conn:           conn,
		subscribeScope: scope,
		send:           make(chan interface{}, 10),
		lastHeartbeat:  time.Now(),
	}
}

// shouldReceive reports whether this client should receive a message with the
// given scope (REQ-WS-05).
func (c *Client) shouldReceive(scope string) bool {
	if scope == "all" {
		return true
	}
	return c.subscribeScope == "all" || c.subscribeScope == scope
}

// ReadPump reads frames from the WebSocket (pong/ping/close handling).
//
// P0 STUB: not implemented. P4 reads via wsjson, updates lastHeartbeat on pong,
// replies pong to client ping, and unregisters on any read error.
//
// TODO(P4): implement the read loop with read deadline / read limit.
func (c *Client) ReadPump() {
	// no-op stub
}

// WritePump drains c.send to the WebSocket and sends periodic pings.
//
// P0 STUB: not implemented. P4 marshals messages, writes with a deadline, and
// emits a ping on a ticker.
//
// TODO(P4): implement the write loop with 30s heartbeat ticker.
func (c *Client) WritePump() {
	// no-op stub
}

// Close unregisters the client and closes the underlying connection.
//
// TODO(P4): close with websocket.StatusGoingAway.
func (c *Client) Close() {
	c.hub.Unregister(c)
}
