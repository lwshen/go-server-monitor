package api

import "github.com/gin-gonic/gin"

// WS upgrades to a WebSocket and subscribes the client (GET /ws?subscribe=all|<id>).
//
// P0 STUB: 501. The coder/websocket accept logic is kept out of the skeleton to
// guarantee a clean compile; the Hub/Client structures it will drive already
// exist in internal/ws.
//
// TODO(P4): validate the subscribe scope ("all" or a 36-char UUID), call
// websocket.Accept(c.Writer, c.Request, ...), build a ws.Client, register it with
// h.deps.Hub, and start its ReadPump/WritePump goroutines.
func (h *Handlers) WS(c *gin.Context) {
	notImplemented(c)
}
