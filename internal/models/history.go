package models

// HistoryPoint is one downsampled bucket returned by GET /api/history
// (REQ-RES-03). Numeric metrics are bucket averages; ping_*/loss_* are nullable
// (nil when every sample in the bucket was unmeasured — AVG over NULLs is NULL).
//
// Ts is the bucket-aligned Unix-seconds timestamp.
type HistoryPoint struct {
	Ts int64 `json:"ts"`

	Cpu         float64 `json:"cpu"`
	MemoryUsed  float64 `json:"memory_used"`
	MemoryTotal float64 `json:"memory_total"`
	SwapUsed    float64 `json:"swap_used"`
	SwapTotal   float64 `json:"swap_total"`
	HddUsed     float64 `json:"hdd_used"`
	HddTotal    float64 `json:"hdd_total"`

	NetworkRx float64 `json:"network_rx"` // B/s
	NetworkTx float64 `json:"network_tx"` // B/s

	TcpConn   float64 `json:"tcp_conn"`
	Processes float64 `json:"processes"`

	PingCt *float64 `json:"ping_ct"`
	PingCu *float64 `json:"ping_cu"`
	PingCm *float64 `json:"ping_cm"`
	PingBd *float64 `json:"ping_bd"`
	LossCt *float64 `json:"loss_ct"`
	LossCu *float64 `json:"loss_cu"`
	LossCm *float64 `json:"loss_cm"`
	LossBd *float64 `json:"loss_bd"`
}
