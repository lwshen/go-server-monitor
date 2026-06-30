package store

import "github.com/uptrace/bun"

// schemaVersion is the current DB schema version baked into the binary
// (REQ-RES-09). Migrate records it in settings.schema_version; P1 will use it to
// drive additive, idempotent migrations.
const schemaVersion = 1

// tableModels lists every persisted table, in creation order. Bun derives
// dialect-correct DDL from these structs, which is what gives multi-database
// support for free across SQLite / libSQL / PostgreSQL.
func tableModels() []any {
	return []any{
		(*serverRow)(nil),
		(*metricRow)(nil),
		(*settingRow)(nil),
	}
}

// serverRow is the `servers` table: editable per-server metadata (REQ-DB-02).
// Timestamps are Unix seconds (CONVENTIONS §1).
type serverRow struct {
	bun.BaseModel `bun:"table:servers,alias:srv"`

	ID              string `bun:"id,pk"`
	Name            string `bun:"name,notnull"`
	ServerGroup     string `bun:"server_group"`
	Gid             string `bun:"gid"`
	Alias           string `bun:"alias"`
	Type            string `bun:"type"`
	Location        string `bun:"location"`
	Weight          int    `bun:"weight"`
	Notify          bool   `bun:"notify"`
	Price           string `bun:"price"`
	ExpireDate      string `bun:"expire_date"` // YYYY-MM-DD, "" = never
	SortOrder       int    `bun:"sort_order"`
	IsHidden        string `bun:"is_hidden"` // "0"/"1"
	CollectInterval int    `bun:"collect_interval"`
	ReportInterval  int    `bun:"report_interval"`
	LastOnlineState string `bun:"last_online_state"` // "online"/"offline" for the offline state machine
	CreatedAt       int64  `bun:"created_at"`        // Unix seconds
	UpdatedAt       int64  `bun:"updated_at"`        // Unix seconds
}

// metricRow is the `metrics_history` table: one row per sample, full retention
// (REQ-DB-03 / REQ-RES-04). Nullable ping_*/loss_*/gpu use pointers so an
// unmeasured value (wire -1) stores as SQL NULL (CONVENTIONS §2).
type metricRow struct {
	bun.BaseModel `bun:"table:metrics_history,alias:mh"`

	ID        int64  `bun:"id,pk,autoincrement"`
	ServerID  string `bun:"server_id,notnull"`
	Timestamp int64  `bun:"timestamp,notnull"` // Unix seconds

	// resource / load
	Cpu       float64 `bun:"cpu"`
	LoadAvg   string  `bun:"load_avg"` // "l1 l5 l15"
	Processes int64   `bun:"processes"`
	TcpConn   int64   `bun:"tcp_conn"`
	UdpConn   int64   `bun:"udp_conn"`
	Thread    int64   `bun:"thread"`

	// memory / swap (MiB)
	MemoryTotal float64 `bun:"memory_total"`
	MemoryUsed  float64 `bun:"memory_used"`
	SwapTotal   float64 `bun:"swap_total"`
	SwapUsed    float64 `bun:"swap_used"`

	// disk aggregate (MiB)
	HddTotal float64 `bun:"hdd_total"`
	HddUsed  float64 `bun:"hdd_used"`

	// network (bytes / B/s)
	NetworkRx   float64 `bun:"network_rx"`
	NetworkTx   float64 `bun:"network_tx"`
	NetworkIn   float64 `bun:"network_in"`
	NetworkOut  float64 `bun:"network_out"`
	NetInSpeed  float64 `bun:"net_in_speed"`
	NetOutSpeed float64 `bun:"net_out_speed"`

	// network quality — nil = unmeasured (SQL NULL)
	PingCt *int64   `bun:"ping_ct"`
	PingCu *int64   `bun:"ping_cu"`
	PingCm *int64   `bun:"ping_cm"`
	PingBd *int64   `bun:"ping_bd"`
	LossCt *float64 `bun:"loss_ct"`
	LossCu *float64 `bun:"loss_cu"`
	LossCm *float64 `bun:"loss_cm"`
	LossBd *float64 `bun:"loss_bd"`

	// GPU (nullable, REQ-RES-07)
	Gpu     *float64 `bun:"gpu"`
	GpuInfo *string  `bun:"gpu_info"`

	// misc
	Uptime    int64  `bun:"uptime"`
	Region    string `bun:"region"` // server-injected ISO country code
	DisksJSON string `bun:"disks_json"`
}

// settingRow is the `settings` key/value table (REQ-RES-01).
type settingRow struct {
	bun.BaseModel `bun:"table:settings,alias:st"`

	Key   string `bun:"key,pk"`
	Value string `bun:"value,notnull"`
}
