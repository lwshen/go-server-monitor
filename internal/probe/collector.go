package probe

import (
	"context"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	psnet "github.com/shirou/gopsutil/v4/net"

	"github.com/lwshen/go-server-monitor/internal/models"
)

const mib = 1024 * 1024 // bytes per MiB

// Collector gathers host metrics into a StatReport via gopsutil (REQ-PROBE-06).
// It is cross-platform (Linux in production, macOS/Windows for dev) and holds the
// state needed to derive rates (network speed, CPU %) between samples.
type Collector struct {
	cfg *Config
	net *NetMonitor

	mu          sync.Mutex
	primed      bool
	prevRx      uint64
	prevTx      uint64
	prevNetTime time.Time

	sysInfo *models.SysInfo // cached; refreshed lazily
}

// NewCollector creates a Collector and primes the CPU-percent baseline so the
// first real sample reflects usage since startup rather than since boot.
func NewCollector(cfg *Config) *Collector {
	_, _ = cpu.Percent(0, false) // prime: discard the since-boot reading
	return &Collector{cfg: cfg, net: NewNetMonitor(cfg.Targets)}
}

// ProbeNetwork runs the three-network + Baidu quality probes (called on the 30s
// ticker); results are merged into subsequent samples.
func (c *Collector) ProbeNetwork() { c.net.Probe() }

// Collect samples the host once (REQ-PROBE-06). Units follow the frozen
// conventions: memory/swap/disk aggregate = MiB, disks[] detail = bytes, network
// cumulative = bytes, network_rx/tx speed = B/s, ping = ms, timestamp = Unix seconds.
func (c *Collector) Collect() (*models.StatReport, error) {
	now := time.Now()
	r := &models.StatReport{
		Frame:    "data",
		LatestTs: now.Unix(),
		Online4:  true,
		SysInfo:  c.systemInfo(),
	}

	// CPU % since the previous call.
	if pct, err := cpu.Percent(0, false); err == nil && len(pct) > 0 {
		r.Cpu = round2(pct[0])
	}

	// Load average.
	if la, err := load.Avg(); err == nil {
		r.Load1, r.Load5, r.Load15 = round2(la.Load1), round2(la.Load5), round2(la.Load15)
	}

	// Memory / swap (MiB).
	if vm, err := mem.VirtualMemory(); err == nil {
		r.MemoryTotal = bytesToMiB(vm.Total)
		r.MemoryUsed = bytesToMiB(vm.Used)
	}
	if sw, err := mem.SwapMemory(); err == nil {
		r.SwapTotal = bytesToMiB(sw.Total)
		r.SwapUsed = bytesToMiB(sw.Used)
	}

	// Disk: aggregate (MiB) + per-filesystem detail (bytes).
	c.collectDisks(r)

	// Network: cumulative bytes + derived speed (B/s).
	c.collectNetwork(r, now)

	// Connections / processes.
	c.collectCounts(r)

	// Uptime.
	if ut, err := host.Uptime(); err == nil {
		r.Uptime = int64(ut)
	}

	// Three-network + Baidu quality (-1 when unmeasured).
	pc, lc := c.net.Get("ct")
	r.PingCt, r.LossCt = float64(pc), float64(lc)
	pc, lc = c.net.Get("cu")
	r.PingCu, r.LossCu = float64(pc), float64(lc)
	pc, lc = c.net.Get("cm")
	r.PingCm, r.LossCm = float64(pc), float64(lc)
	pc, lc = c.net.Get("bd")
	r.PingBd, r.LossBd = float64(pc), float64(lc)

	return r, nil
}

func (c *Collector) collectDisks(r *models.StatReport) {
	parts, err := disk.Partitions(false)
	if err != nil {
		return
	}
	seen := map[string]bool{} // dedupe by mountpoint
	var totalB, usedB uint64
	for _, p := range parts {
		if seen[p.Mountpoint] {
			continue
		}
		seen[p.Mountpoint] = true
		u, err := disk.Usage(p.Mountpoint)
		if err != nil || u.Total == 0 {
			continue
		}
		totalB += u.Total
		usedB += u.Used
		r.Disks = append(r.Disks, models.DiskInfo{
			Name:       p.Device,
			MountPoint: p.Mountpoint,
			FileSystem: p.Fstype,
			Total:      float64(u.Total),
			Used:       float64(u.Used),
			Free:       float64(u.Free),
		})
	}
	r.HddTotal = bytesToMiB(totalB)
	r.HddUsed = bytesToMiB(usedB)
}

func (c *Collector) collectNetwork(r *models.StatReport, now time.Time) {
	counters, err := psnet.IOCounters(false) // false = aggregate over all NICs
	if err != nil || len(counters) == 0 {
		return
	}
	rx, tx := counters[0].BytesRecv, counters[0].BytesSent
	r.NetworkIn, r.NetworkOut = float64(rx), float64(tx)

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.primed {
		if elapsed := now.Sub(c.prevNetTime).Seconds(); elapsed > 0 {
			r.NetworkRx = rate(rx, c.prevRx, elapsed)
			r.NetworkTx = rate(tx, c.prevTx, elapsed)
		}
	}
	c.prevRx, c.prevTx, c.prevNetTime, c.primed = rx, tx, now, true
}

func (c *Collector) collectCounts(r *models.StatReport) {
	// Connection counts are best-effort: on some platforms they need privileges
	// or shell-outs, so failures degrade to 0 rather than aborting the sample.
	if conns, err := psnet.Connections("tcp"); err == nil {
		r.Tcp = int64(len(conns))
	}
	if conns, err := psnet.Connections("udp"); err == nil {
		r.Udp = int64(len(conns))
	}
	if info, err := host.Info(); err == nil {
		r.Process = int64(info.Procs)
	}
}

// systemInfo builds (and caches) the host SysInfo snapshot.
func (c *Collector) systemInfo() *models.SysInfo {
	c.mu.Lock()
	if c.sysInfo != nil {
		si := c.sysInfo
		c.mu.Unlock()
		return si
	}
	c.mu.Unlock()

	si := &models.SysInfo{}
	if info, err := host.Info(); err == nil {
		si.Name = info.Platform
		si.Version = info.PlatformVersion
		si.OsName = info.OS
		si.OsArch = info.KernelArch
		si.OsFamily = info.PlatformFamily
		si.OsRelease = info.KernelVersion
		si.KernelVersion = info.KernelVersion
		si.HostName = info.Hostname
	}
	if n, err := cpu.CountsWithContext(context.Background(), true); err == nil {
		si.CpuNum = n
	}
	if ci, err := cpu.Info(); err == nil && len(ci) > 0 {
		si.CpuBrand = ci[0].ModelName
		si.CpuVenderId = ci[0].VendorID
	}

	c.mu.Lock()
	c.sysInfo = si
	c.mu.Unlock()
	return si
}

func bytesToMiB(b uint64) float64 { return round2(float64(b) / mib) }

func rate(cur, prev uint64, elapsed float64) float64 {
	if cur < prev { // counter wrap/reset
		return 0
	}
	return round2(float64(cur-prev) / elapsed)
}

func round2(f float64) float64 { return float64(int64(f*100+0.5)) / 100 }
