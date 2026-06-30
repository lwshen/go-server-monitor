package probe

import (
	"time"

	"github.com/lwshen/go-server-monitor/internal/models"
)

// Collector gathers system metrics into a StatReport (REQ-PROBE-06).
type Collector struct {
	cfg *Config
}

// NewCollector creates a Collector bound to cfg.
func NewCollector(cfg *Config) *Collector {
	return &Collector{cfg: cfg}
}

// Collect samples the host once, returning a StatReport (REQ-PROBE-06).
//
// P0 STUB: returns a zeroed report with only frame/latest_ts set. Real collection
// (CPU/mem/disk/net/load/connections/sysinfo) lands in P3.
//
// TODO(P3): read /proc (or syscalls) for cpu/mem/disk/net/load/processes; fill
// SysInfo; convert memory/swap/disk to MiB and net to bytes / B/s per conventions.
func (c *Collector) Collect() (*models.StatReport, error) {
	return &models.StatReport{
		Frame:    "data",
		LatestTs: time.Now().Unix(),
	}, nil
}
