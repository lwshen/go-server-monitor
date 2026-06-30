// Package models defines the Go structs mirroring the canonical wire contract.
//
// The single source of truth for the report contract is requirements/report-types.ts
// (REQ-RES-00). The structs below mirror that file field-for-field, with json tags
// matching the .ts names exactly.
//
// Units / sentinels (frozen, CONVENTIONS.md §1-2):
//   - memory/swap/disk aggregate = MiB; disks[] detail = bytes
//   - net cumulative = bytes; net speed = B/s
//   - timestamps (latest_ts) = Unix SECONDS
//   - ping = ms; cpu/loss = % (0-100)
//   - unmeasured numeric -> -1 on the wire (mainly ping_*/loss_*)
package models

// StatReport is the main message a probe sends each report cycle.
// Mirrors StatReport in report-types.ts.
type StatReport struct {
	// ── identity / metadata ─────────────────────────────
	Name     string `json:"name"`      // host identifier
	Version  string `json:"version"`   // client version
	LatestTs int64  `json:"latest_ts"` // report timestamp (Unix seconds)
	Frame    string `json:"frame"`     // frame/mode marker, fixed "data"

	// ── traffic source / connectivity ───────────────────
	Vnstat  bool `json:"vnstat"`  // traffic sourced from vnstat
	Online4 bool `json:"online4"` // IPv4 reachable
	Online6 bool `json:"online6"` // IPv6 reachable

	Uptime int64 `json:"uptime"` // uptime (seconds)

	// ── load (/proc/loadavg) ────────────────────────────
	Load1  float64 `json:"load_1"`
	Load5  float64 `json:"load_5"`
	Load15 float64 `json:"load_15"`

	// ── network quality (CF naming): ct/cu/cm/bd ─────────
	// ping = ms, loss = % (0-100); -1 = unmeasured.
	PingCt float64 `json:"ping_ct"` // China Telecom latency ms
	PingCu float64 `json:"ping_cu"` // China Unicom latency ms
	PingCm float64 `json:"ping_cm"` // China Mobile latency ms
	PingBd float64 `json:"ping_bd"` // Baidu latency ms
	LossCt float64 `json:"loss_ct"` // China Telecom loss %
	LossCu float64 `json:"loss_cu"` // China Unicom loss %
	LossCm float64 `json:"loss_cm"` // China Mobile loss %
	LossBd float64 `json:"loss_bd"` // Baidu loss %

	// ── connections / processes ─────────────────────────
	Tcp     int64 `json:"tcp"`     // TCP connection count
	Udp     int64 `json:"udp"`     // UDP socket count
	Process int64 `json:"process"` // process count
	Thread  int64 `json:"thread"`  // thread count

	// ── network ─────────────────────────────────────────
	NetworkRx      float64 `json:"network_rx"`       // instantaneous downlink speed (B/s)
	NetworkTx      float64 `json:"network_tx"`       // instantaneous uplink speed (B/s)
	NetworkIn      float64 `json:"network_in"`       // cumulative inbound (bytes)
	NetworkOut     float64 `json:"network_out"`      // cumulative outbound (bytes)
	LastNetworkIn  float64 `json:"last_network_in"`  // period inbound (bytes, vnstat mode)
	LastNetworkOut float64 `json:"last_network_out"` // period outbound (bytes, vnstat mode)

	// ── CPU / memory / disk ─────────────────────────────
	Cpu         float64 `json:"cpu"`          // CPU usage %
	MemoryTotal float64 `json:"memory_total"` // total memory (MiB)
	MemoryUsed  float64 `json:"memory_used"`  // used memory (MiB)
	SwapTotal   float64 `json:"swap_total"`   // total swap (MiB)
	SwapUsed    float64 `json:"swap_used"`    // used swap (MiB)
	HddTotal    float64 `json:"hdd_total"`    // total disk (MiB)
	HddUsed     float64 `json:"hdd_used"`     // used disk (MiB)

	// ── GPU (optional, REQ-RES-07) ──────────────────────
	Gpu     *float64 `json:"gpu,omitempty"`      // GPU usage %, nil if not collected
	GpuInfo string   `json:"gpu_info,omitempty"` // GPU model string

	Custom string `json:"custom,omitempty"` // optional custom field

	SysInfo *SysInfo `json:"sys_info,omitempty"` // optional host system info
	IpInfo  *IpInfo  `json:"ip_info,omitempty"`  // optional IP geo info

	// ── grouping / display metadata (probe-pushed) ──────
	Gid      string `json:"gid"`      // group id
	Alias    string `json:"alias"`    // display alias
	Weight   int    `json:"weight"`   // sort weight
	Type     string `json:"type"`     // host type label
	Location string `json:"location"` // location label
	Notify   bool   `json:"notify"`   // whether to notify for this host
	Si       bool   `json:"si"`       // unit system: false=binary(MiB) / true=decimal(MB)

	Disks []DiskInfo `json:"disks"` // per-filesystem detail
}

// SysInfo holds host system information. Mirrors SysInfo in report-types.ts.
type SysInfo struct {
	Name          string `json:"name"`       // distro name
	Version       string `json:"version"`    // OS version
	OsName        string `json:"os_name"`    // OS name
	OsArch        string `json:"os_arch"`    // CPU architecture
	OsFamily      string `json:"os_family"`  // e.g. unix
	OsRelease     string `json:"os_release"` // OS release string
	KernelVersion string `json:"kernel_version"`
	CpuNum        int    `json:"cpu_num"`       // CPU core count
	CpuBrand      string `json:"cpu_brand"`     // CPU model
	CpuVenderId   string `json:"cpu_vender_id"` // CPU vendor id
	HostName      string `json:"host_name"`     // host name
}

// DiskInfo is a single filesystem's detail (bytes). Mirrors DiskInfo in report-types.ts.
type DiskInfo struct {
	Name       string  `json:"name"`        // device name
	MountPoint string  `json:"mount_point"` // mount point
	FileSystem string  `json:"file_system"` // filesystem type
	Total      float64 `json:"total"`       // total size (bytes)
	Used       float64 `json:"used"`        // used (bytes)
	Free       float64 `json:"free"`        // free (bytes)
}

// IpInfo holds IP geo / ISP information. Mirrors IpInfo in report-types.ts.
type IpInfo struct {
	Query      string  `json:"query"`  // queried IP
	Source     string  `json:"source"` // data source, e.g. ip-api
	Continent  string  `json:"continent"`
	Country    string  `json:"country"`
	RegionName string  `json:"region_name"` // province/region
	City       string  `json:"city"`
	Isp        string  `json:"isp"`
	Org        string  `json:"org"`
	As         string  `json:"as"`     // AS number/string
	AsName     string  `json:"asname"` // AS org name
	Lat        float64 `json:"lat"`    // latitude
	Lon        float64 `json:"lon"`    // longitude
	Timezone   string  `json:"timezone"`
}

// ReportRequest is the wire envelope a probe POSTs to /report.
//
// The envelope shape is debated between chapters; report-types.ts (StatReport)
// is the canonical body contract. Per REQ-RES-05 the probe authenticates with a
// plaintext "secret" field in the JSON body (no HMAC). When collect_interval <
// report_interval a single POST carries multiple Samples (REQ-RES-04).
//
// TODO(P2): reconcile the exact envelope vs. flat-StatReport debate and finalize
// how Samples relate to the top-level StatReport fields.
type ReportRequest struct {
	ID      string        `json:"id"`                // server id
	Secret  string        `json:"secret"`            // plaintext shared secret (constant-time compared)
	Report  *StatReport   `json:"report,omitempty"`  // single-sample report (top-level metrics)
	Samples []*StatReport `json:"samples,omitempty"` // multi-sample batch (each with its own latest_ts)
}
