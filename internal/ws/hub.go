package ws

import (
	"sync"
	"time"

	"github.com/lwshen/go-server-monitor/internal/models"
	"go.uber.org/zap"
)

// Hub is the in-process WebSocket connection manager and broadcast center
// (REQ-WS-01). It replaces the Cloudflare Durable Object with a goroutine +
// channel design.
type Hub struct {
	register   chan *Client               // new connection registration
	unregister chan *Client               // disconnect / deregistration
	broadcast  chan *models.BroadcastData // messages awaiting fan-out
	clients    map[*Client]bool           // active connections
	mu         sync.RWMutex               // guards clients
	done       chan struct{}              // closed by Stop()
	log        *zap.Logger
}

// NewHub creates a Hub with the buffered control-plane channels from the spec
// (register/unregister=100, broadcast=500).
func NewHub(log *zap.Logger) *Hub {
	return &Hub{
		register:   make(chan *Client, 100),
		unregister: make(chan *Client, 100),
		broadcast:  make(chan *models.BroadcastData, 500),
		clients:    make(map[*Client]bool),
		done:       make(chan struct{}),
		log:        log,
	}
}

// Run is the Hub's main event loop; call it in a goroutine (`go hub.Run()`).
//
// The select loop over register/unregister/broadcast/done is real wiring. The
// actual per-message fan-out to WebSocket connections is a P4 stub.
//
// TODO(P4): on register send HelloMessage; on broadcast filter by scope and do
// non-blocking sends (select+default) with slow-consumer drop counting.
func (h *Hub) Run() {
	for {
		select {
		case <-h.done:
			h.closeAll()
			return
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = true
			h.mu.Unlock()
			// Greet the new subscriber (REQ-WS-04.1).
			c.enqueue(models.WsMessage{Type: "hello", Ts: time.Now().Unix(), Subscribed: c.subscribeScope})
		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
			h.mu.Unlock()
		case msg := <-h.broadcast:
			h.fanout(msg)
		}
	}
}

// Broadcast queues a message for fan-out without blocking the caller; if the
// broadcast channel is full the message is dropped (REQ-WS-06 background pressure).
//
// TODO(P4): called from service.SaveMetrics after a successful write.
func (h *Hub) Broadcast(msg *models.BroadcastData) {
	select {
	case h.broadcast <- msg:
	default:
		h.log.Warn("ws hub broadcast queue full, message dropped")
	}
}

// fanout delivers a message to every client subscribed to its scope, using a
// non-blocking enqueue so a slow consumer is skipped rather than blocking the Hub
// (REQ-WS-03). The wire frame is built once and shared (read-only).
func (h *Hub) fanout(msg *models.BroadcastData) {
	frame := toFrame(msg)
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		if !c.shouldReceive(msg.Scope) {
			continue
		}
		select {
		case c.send <- frame:
		default:
			h.log.Warn("ws slow consumer, frame dropped", zap.String("scope", c.subscribeScope))
		}
	}
}

// toFrame converts an internal BroadcastData into the wire WsMessage: a single
// sample is an "update", multiple samples a "batchUpdate" (REQ-RES-04).
func toFrame(msg *models.BroadcastData) models.WsMessage {
	if msg.Type == "batchUpdate" {
		return models.WsMessage{
			Type:    "batchUpdate",
			Ts:      msg.Ts,
			Updates: []models.BatchUpdate{{ServerID: msg.ServerID, Samples: msg.Samples}},
		}
	}
	return models.WsMessage{Type: "update", ServerID: msg.ServerID, Ts: msg.Ts, Data: msg.Data}
}

// Register enqueues a client for registration (used by the /ws upgrade handler).
func (h *Hub) Register(c *Client) { h.register <- c }

// Unregister enqueues a client for removal.
func (h *Hub) Unregister(c *Client) { h.unregister <- c }

// closeAll closes every client's send channel and resets the pool.
func (h *Hub) closeAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		close(c.send)
	}
	h.clients = make(map[*Client]bool)
}

// Stop signals the Run loop to close all connections and exit.
func (h *Hub) Stop() {
	close(h.done)
}
