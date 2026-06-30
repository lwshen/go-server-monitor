// Package ws implements the in-process WebSocket broadcast Hub (REQ-WS-01..13).
// P0 provides the structures and the Run() select-loop skeleton; the real
// coder/websocket accept/read/write logic lands in P4.
package ws

// HelloMessage is sent to a client immediately after it registers (REQ-WS-04.1).
type HelloMessage struct {
	Type       string `json:"type"`       // "hello"
	Ts         int64  `json:"ts"`         // Unix seconds
	Subscribed string `json:"subscribed"` // "all" or a serverId
}

// UpdateMessage is a single-server realtime update (REQ-WS-04.2).
type UpdateMessage struct {
	Type     string         `json:"type"`     // "update"
	ServerID string         `json:"serverId"` // server UUID
	Ts       int64          `json:"ts"`       // Unix seconds
	Data     map[string]any `json:"data"`     // dynamic-only metrics
}

// BatchUpdateMessage merges multiple servers/samples into one frame (REQ-WS-04.3).
type BatchUpdateMessage struct {
	Type    string        `json:"type"` // "batchUpdate"
	Ts      int64         `json:"ts"`   // Unix seconds
	Updates []ServerBatch `json:"updates"`
}

// ServerBatch is one server's samples inside a BatchUpdateMessage.
type ServerBatch struct {
	ServerID string        `json:"serverId"`
	Samples  []SampleFrame `json:"samples"`
}

// SampleFrame is a timestamped data point inside a ServerBatch.
type SampleFrame struct {
	Ts   int64          `json:"ts"`
	Data map[string]any `json:"data"`
}

// PingMessage / PongMessage are the heartbeat frames (REQ-WS-04.4).
type PingMessage struct {
	Type string `json:"type"` // "ping"
}

type PongMessage struct {
	Type string `json:"type"` // "pong"
}
