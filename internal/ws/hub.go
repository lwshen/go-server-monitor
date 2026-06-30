package ws

import (
	"sync"

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

// fanout delivers a message to matching clients.
//
// P0 STUB: iterates clients but does not yet serialize/write — that is P4.
func (h *Hub) fanout(msg *models.BroadcastData) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	// TODO(P4): for each client, if shouldReceive(msg.Scope) do a non-blocking
	// send to c.send; count drops for slow consumers.
	_ = msg
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
