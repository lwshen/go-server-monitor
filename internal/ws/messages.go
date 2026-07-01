// Package ws implements the in-process WebSocket broadcast Hub (REQ-WS-01..13):
// a goroutine + channel design that replaces the Cloudflare Durable Object.
//
// The wire frame is models.WsMessage for every message type — "hello" on connect,
// "update" / "batchUpdate" on ingest (REQ-RES-04), and "ping"/"pong" heartbeats.
// The frame carries only dynamic metrics (models.DynamicData, REQ-RES-06); static
// fields are fetched via the /api/servers and /api/server snapshots.
package ws
