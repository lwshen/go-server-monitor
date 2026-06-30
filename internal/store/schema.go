package store

import "github.com/uptrace/bun"

// This file is the SINGLE SOURCE OF TRUTH for the database schema. Bun derives
// dialect-correct DDL from these structs (SQLite / libSQL / PostgreSQL), so the
// columns here must mirror the authoritative report contract:
//
//	requirements/report-types.ts  +  internal/models.{MetricsRow,ServerConfig}
//
// Keep migrations/schema.sql (a human-readable reference) in sync with this file.
// Frozen facts (CONVENTIONS.md / 14-resolved-decisions.md):
//   - timestamp = Unix SECONDS; memory/swap/disk aggregate = MiB; net cumulative
//     = bytes; net speed (network_rx/tx) = B/s.
//   - unmeasured ping_*/loss_*/gpu arrive as -1 on the wire and are stored as SQL
//     NULL — hence the pointer fields (nil = "no data", renders "—").
//   - disks[] detail is stored as disks_json TEXT (REQ-RES-02), no relational table.

// serverRow is the `servers` table: per-server config + display metadata
// (mirrors models.ServerConfig) plus DB-only state used by the P7 jobs and the
// latest sys_info / ip_info JSON snapshots (REQ-RES-06: static, fetched via the
// snapshot APIs, never broadcast).
type serverRow struct {
	bun.BaseModel `bun:"table:servers,alias:srv"`

	ID              string `bun:"id,pk"`
	Name            string `bun:"name,notnull"`
	ServerGroup     string `bun:"server_group,default:'Default'"`
	Price           string `bun:"price"`                             // free-text monthly price
	ExpireDate      string `bun:"expire_date"`                       // YYYY-MM-DD, "" = never
	Bandwidth       string `bun:"bandwidth"`                         // free-text bandwidth tier
	TrafficLimit    string `bun:"traffic_limit"`                     // free-text monthly cap
	TrafficCalcType string `bun:"traffic_calc_type,default:'total'"` // total/up/down/max
	ResetDay        int    `bun:"reset_day,default:1"`               // monthly reset day (1-31)
	CollectInterval int    `bun:"collect_interval"`                  // desired sample interval (s); 0 = none
	ReportInterval  int    `bun:"report_interval,default:60"`        // desired report interval (s)
	PingMode        string `bun:"ping_mode,default:'http'"`          // http/tcp
	IsHidden        string `bun:"is_hidden,default:'0'"`             // "0" visible / "1" hidden
	SortOrder       int    `bun:"sort_order"`                        // ascending sort weight

	// offline-detection / expiration-reminder state (P7)
	LastOnlineState    int   `bun:"last_online_state,default:1"` // 1 online / 0 offline
	LastStateChange    int64 `bun:"last_state_change"`           // Unix seconds of last transition
	ExpirationNotified int   `bun:"expiration_notified"`         // 0/1

	// latest structured snapshots (refreshed on report; ip_info carries lat/lon for the map)
	SysInfoJSON string `bun:"sys_info_json"`
	IpInfoJSON  string `bun:"ip_info_json"`

	CreatedAt string `bun:"created_at"` // UTC ISO 8601
	UpdatedAt string `bun:"updated_at"` // UTC ISO 8601
}

// metricRow is the `metrics_history` table: one row per reported sample, full
// retention (REQ-DB-03 / REQ-RES-04). Mirrors models.MetricsRow field-for-field
// so the P2 mapping is 1:1.
type metricRow struct {
	bun.BaseModel `bun:"table:metrics_history,alias:mh"`

	ID        int64  `bun:"id,pk,autoincrement"` // AUTOINCREMENT (SQLite) / BIGSERIAL (Postgres)
	ServerID  string `bun:"server_id,notnull"`
	Timestamp int64  `bun:"timestamp,notnull"` // Unix SECONDS

	// resource / load / process
	Cpu       float64 `bun:"cpu"`      // %
	LoadAvg   string  `bun:"load_avg"` // "l1 l5 l15"
	Processes int64   `bun:"processes"`
	TcpConn   int64   `bun:"tcp_conn"`
	UdpConn   int64   `bun:"udp_conn"`
	Thread    int64   `bun:"thread"`
	CpuCores  int     `bun:"cpu_cores"`
	CpuInfo   string  `bun:"cpu_info"`
	CpuModel  string  `bun:"cpu_model"`

	// memory / swap (MiB)
	MemoryTotal float64 `bun:"memory_total"`
	MemoryUsed  float64 `bun:"memory_used"`
	SwapTotal   float64 `bun:"swap_total"`
	SwapUsed    float64 `bun:"swap_used"`

	// disk aggregate (MiB)
	HddTotal float64 `bun:"hdd_total"`
	HddUsed  float64 `bun:"hdd_used"`

	// network: rx/tx are instantaneous speed (B/s); in/out are cumulative (bytes)
	NetworkRx      float64 `bun:"network_rx"`
	NetworkTx      float64 `bun:"network_tx"`
	NetworkIn      float64 `bun:"network_in"`
	NetworkOut     float64 `bun:"network_out"`
	LastNetworkIn  float64 `bun:"last_network_in"`
	LastNetworkOut float64 `bun:"last_network_out"`

	// network quality — nil = unmeasured (SQL NULL); ping in ms, loss in %
	PingCt *int64   `bun:"ping_ct"`
	PingCu *int64   `bun:"ping_cu"`
	PingCm *int64   `bun:"ping_cm"`
	PingBd *int64   `bun:"ping_bd"`
	LossCt *float64 `bun:"loss_ct"`
	LossCu *float64 `bun:"loss_cu"`
	LossCm *float64 `bun:"loss_cm"`
	LossBd *float64 `bun:"loss_bd"`

	// IP reachability (0/1)
	Online4 int `bun:"online4"`
	Online6 int `bun:"online6"`

	// system info (flattened per-sample; full structured snapshot lives on servers)
	Os            string `bun:"os"`
	OsRelease     string `bun:"os_release"`
	KernelVersion string `bun:"kernel_version"`
	Arch          string `bun:"arch"`
	OsFamily      string `bun:"os_family"`
	Uptime        int64  `bun:"uptime"` // seconds
	HostName      string `bun:"host_name"`

	// GPU (nullable, REQ-RES-07)
	Gpu     *float64 `bun:"gpu"`
	GpuInfo string   `bun:"gpu_info"`

	// geo / grouping
	Region   string `bun:"region"` // server-injected ISO country code
	Gid      string `bun:"gid"`
	Location string `bun:"location"`

	// metadata
	Vnstat int    `bun:"vnstat"` // 0/1
	Custom string `bun:"custom"`

	// disk detail as a JSON array string (REQ-RES-02)
	DisksJSON string `bun:"disks_json"`
}

// settingRow is the `settings` key/value table (REQ-RES-01).
type settingRow struct {
	bun.BaseModel `bun:"table:settings,alias:st"`

	Key   string `bun:"key,pk"`
	Value string `bun:"value,notnull"`
}
