package models

// WsMessage is a WebSocket frame sent server -> client.
//
// Message types (05-realtime-websocket.md): "hello", "update", "batchUpdate",
// "ping", "pong". Timestamps are Unix seconds per the frozen conventions
// (CONVENTIONS.md §1) — note the chapter sketches show ms, but seconds is
// authoritative.
type WsMessage struct {
	Type       string         `json:"type"`                 // hello/update/batchUpdate/ping/pong
	ServerID   string         `json:"serverId,omitempty"`   // for update frames
	Ts         int64          `json:"ts,omitempty"`         // Unix seconds
	Subscribed string         `json:"subscribed,omitempty"` // hello: echoed scope
	Data       map[string]any `json:"data,omitempty"`       // dynamic-only metrics (update)
	Updates    []BatchUpdate  `json:"updates,omitempty"`    // batchUpdate payload
}

// BatchUpdate is one server's bundle of samples within a batchUpdate frame (REQ-RES-04).
type BatchUpdate struct {
	ServerID string        `json:"serverId"`
	Samples  []BatchSample `json:"samples"`
}

// BatchSample is a single timestamped data point inside a BatchUpdate.
type BatchSample struct {
	Ts   int64          `json:"ts"`   // Unix seconds
	Data map[string]any `json:"data"` // dynamic-only metrics
}

// BroadcastData is the internal message wrapper queued onto the Hub's broadcast
// channel before fan-out (the "type, serverId, ts, data" envelope).
type BroadcastData struct {
	Type     string         // "update" or "batchUpdate"
	Scope    string         // "all" or a serverId
	Ts       int64          // Unix seconds
	ServerID string         // target server
	Data     map[string]any // dynamic-only metrics
}

// BroadcastDeleteFields is the static-field exclusion set (REQ-RES-06): these
// low-frequency / static fields are NOT included in broadcast `data`. The
// frontend obtains them from the /api/servers and /api/server snapshots; the
// realtime stream only carries the dynamic metrics.
//
// TODO(P4): apply this set in service.SaveMetrics when constructing broadcast data.
var BroadcastDeleteFields = map[string]bool{
	"name":       true,
	"alias":      true,
	"gid":        true,
	"weight":     true,
	"type":       true,
	"location":   true,
	"notify":     true,
	"si":         true,
	"version":    true,
	"frame":      true,
	"sys_info":   true,
	"ip_info":    true,
	"disks":      true,
	"disks_json": true,
	"uptime":     true, // optional
	"latest_ts":  true, // replaced by frame-level ts
}
