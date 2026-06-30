package models

// MetricsRow mirrors one row of the metrics_history table (REQ-DB-03).
//
// Sentinel handling (CONVENTIONS.md §2): unmeasured ping_*/loss_* arrive as -1 on
// the wire and are stored as SQL NULL — hence the pointer types here, where nil
// means "no data" and renders as "—" on the frontend.
//
// Disk detail is stored as a JSON array string in DisksJSON (REQ-RES-02), not a
// relational table.
//
// Units follow the frozen conventions: memory/swap/disk aggregate = MiB, net
// cumulative = bytes, net speed = B/s, timestamp = Unix seconds.
type MetricsRow struct {
	ID        int64  `json:"-"`
	ServerID  string `json:"server_id"`
	Timestamp int64  `json:"timestamp"` // Unix seconds

	// resource / summary
	Cpu       float64 `json:"cpu"`
	LoadAvg   string  `json:"load_avg"` // "l1 l5 l15"
	Processes int64   `json:"processes"`
	TcpConn   int64   `json:"tcp_conn"`
	UdpConn   int64   `json:"udp_conn"`
	Thread    int64   `json:"thread"`
	CpuCores  int     `json:"cpu_cores"`
	CpuInfo   string  `json:"cpu_info"`
	CpuModel  string  `json:"cpu_model"`

	// memory / swap (MiB)
	MemoryTotal float64 `json:"memory_total"`
	MemoryUsed  float64 `json:"memory_used"`
	SwapTotal   float64 `json:"swap_total"`
	SwapUsed    float64 `json:"swap_used"`

	// disk aggregate (MiB)
	HddTotal float64 `json:"hdd_total"`
	HddUsed  float64 `json:"hdd_used"`

	// network (bytes / B/s)
	NetworkRx      float64 `json:"network_rx"`
	NetworkTx      float64 `json:"network_tx"`
	NetworkIn      float64 `json:"network_in"`
	NetworkOut     float64 `json:"network_out"`
	LastNetworkIn  float64 `json:"last_network_in"`
	LastNetworkOut float64 `json:"last_network_out"`

	// network quality — nil = unmeasured (SQL NULL)
	PingCt *int64   `json:"ping_ct"`
	PingCu *int64   `json:"ping_cu"`
	PingCm *int64   `json:"ping_cm"`
	PingBd *int64   `json:"ping_bd"`
	LossCt *float64 `json:"loss_ct"`
	LossCu *float64 `json:"loss_cu"`
	LossCm *float64 `json:"loss_cm"`
	LossBd *float64 `json:"loss_bd"`

	// IP reachability
	Online4 int `json:"online4"` // 0/1
	Online6 int `json:"online6"` // 0/1

	// system info
	Os            string `json:"os"`
	OsRelease     string `json:"os_release"`
	KernelVersion string `json:"kernel_version"`
	Arch          string `json:"arch"`
	OsFamily      string `json:"os_family"`
	Uptime        int64  `json:"uptime"`
	HostName      string `json:"host_name"`

	// GPU (nullable, REQ-RES-07)
	Gpu     *float64 `json:"gpu"`
	GpuInfo string   `json:"gpu_info"`

	// geo / grouping
	Region   string `json:"region"` // server-injected ISO country code
	Gid      string `json:"gid"`
	Location string `json:"location"`

	// metadata
	Vnstat int    `json:"vnstat"` // 0/1
	Custom string `json:"custom"`

	// disk detail as JSON array string (REQ-RES-02)
	DisksJSON string `json:"disks_json"`
}
