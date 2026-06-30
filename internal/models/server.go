package models

// ServerConfig is the editable metadata for a monitored server (the `servers`
// table, REQ-DB-02). Managed via the admin API; merged with the latest metrics
// for display.
type ServerConfig struct {
	ID              string `json:"id"`                // UUID v4 primary key
	Name            string `json:"name"`              // display name (NOT NULL)
	ServerGroup     string `json:"server_group"`      // grouping label
	Price           string `json:"price"`             // free-text monthly price
	ExpireDate      string `json:"expire_date"`       // YYYY-MM-DD, "" =永久
	Bandwidth       string `json:"bandwidth"`         // free-text bandwidth tier
	TrafficLimit    string `json:"traffic_limit"`     // free-text monthly cap
	TrafficCalcType string `json:"traffic_calc_type"` // total/up/down/max
	ResetDay        int    `json:"reset_day"`         // monthly traffic reset day (1-31)
	CollectInterval int    `json:"collect_interval"`  // desired sample interval (s); 0 = no subsampling
	ReportInterval  int    `json:"report_interval"`   // desired report interval (s)
	PingMode        string `json:"ping_mode"`         // http/tcp
	IsHidden        string `json:"is_hidden"`         // "0" visible / "1" hidden
	SortOrder       int    `json:"sort_order"`        // ascending sort weight
	CreatedAt       string `json:"created_at"`        // UTC ISO 8601
	UpdatedAt       string `json:"updated_at"`        // UTC ISO 8601
}

// Server is a ServerConfig joined with its latest metrics snapshot, as returned
// by GET /api/servers and GET /api/server.
//
// TODO(P2): fill LatestMetrics from the most recent metrics_history row and
// compute online/offline status.
type Server struct {
	ServerConfig
	LatestMetrics *MetricsRow `json:"latest_metrics,omitempty"`
	LastUpdated   int64       `json:"last_updated"` // Unix seconds of latest sample
	Online        bool        `json:"online"`       // derived from offline detection
}
