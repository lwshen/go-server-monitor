package store

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/lwshen/go-server-monitor/internal/models"
)

// msThreshold distinguishes Unix seconds from milliseconds. A seconds timestamp
// stays below ~1e10 until the year 2286; a millisecond timestamp is ~1.7e12.
const msThreshold = int64(1e10)

// timedSample is one resolved (timestamp, payload) pair ready to persist.
type timedSample struct {
	ts   int64
	data *models.StatReport
}

// resolveSamples flattens a report into the rows to write (REQ-RES-04): when a
// batch is present, each entry becomes a row; otherwise the single top-level Data
// is the only row. Timestamps are normalized to Unix seconds.
func resolveSamples(req *models.ReportRequest) []timedSample {
	now := time.Now().Unix()
	if len(req.Samples) > 0 {
		out := make([]timedSample, 0, len(req.Samples))
		for _, s := range req.Samples {
			if s.Data == nil {
				continue
			}
			out = append(out, timedSample{ts: normalizeUnixSeconds(s.Timestamp, s.Data, now), data: s.Data})
		}
		return out
	}
	if req.Data != nil {
		return []timedSample{{ts: normalizeUnixSeconds(req.Timestamp, req.Data, now), data: req.Data}}
	}
	return nil
}

// normalizeUnixSeconds picks the best timestamp (explicit envelope ts, else the
// payload's latest_ts, else server-now) and converts milliseconds to seconds
// (03-report-protocol §2.4.3).
func normalizeUnixSeconds(ts int64, sr *models.StatReport, now int64) int64 {
	cand := ts
	if cand <= 0 && sr != nil {
		cand = sr.LatestTs
	}
	if cand <= 0 {
		return now
	}
	if cand > msThreshold {
		return cand / 1000
	}
	return cand
}

// statReportToMetricRow maps a wire StatReport to a metrics_history row. Sentinel
// handling (CONVENTIONS §2): -1 ping/loss and a negative/absent gpu become SQL
// NULL (nil pointer). Booleans become 0/1; load becomes "l1 l5 l15"; disks are
// JSON-encoded (REQ-RES-02).
func statReportToMetricRow(serverID string, ts int64, sr *models.StatReport) *metricRow {
	m := &metricRow{
		ServerID:       serverID,
		Timestamp:      ts,
		Cpu:            sr.Cpu,
		LoadAvg:        formatLoad(sr.Load1, sr.Load5, sr.Load15),
		Processes:      sr.Process,
		TcpConn:        sr.Tcp,
		UdpConn:        sr.Udp,
		Thread:         sr.Thread,
		MemoryTotal:    sr.MemoryTotal,
		MemoryUsed:     sr.MemoryUsed,
		SwapTotal:      sr.SwapTotal,
		SwapUsed:       sr.SwapUsed,
		HddTotal:       sr.HddTotal,
		HddUsed:        sr.HddUsed,
		NetworkRx:      sr.NetworkRx,
		NetworkTx:      sr.NetworkTx,
		NetworkIn:      sr.NetworkIn,
		NetworkOut:     sr.NetworkOut,
		LastNetworkIn:  sr.LastNetworkIn,
		LastNetworkOut: sr.LastNetworkOut,
		PingCt:         pingPtr(sr.PingCt),
		PingCu:         pingPtr(sr.PingCu),
		PingCm:         pingPtr(sr.PingCm),
		PingBd:         pingPtr(sr.PingBd),
		LossCt:         lossPtr(sr.LossCt),
		LossCu:         lossPtr(sr.LossCu),
		LossCm:         lossPtr(sr.LossCm),
		LossBd:         lossPtr(sr.LossBd),
		Online4:        boolToInt(sr.Online4),
		Online6:        boolToInt(sr.Online6),
		Uptime:         sr.Uptime,
		Gpu:            gpuPtr(sr.Gpu),
		GpuInfo:        sr.GpuInfo,
		Gid:            sr.Gid,
		Location:       sr.Location,
		Vnstat:         boolToInt(sr.Vnstat),
		Custom:         sr.Custom,
		DisksJSON:      disksJSON(sr.Disks),
	}
	if si := sr.SysInfo; si != nil {
		m.Os = si.OsName
		m.OsRelease = si.OsRelease
		m.KernelVersion = si.KernelVersion
		m.Arch = si.OsArch
		m.OsFamily = si.OsFamily
		m.HostName = si.HostName
		m.CpuCores = si.CpuNum
		// cpu_info is the CPU model string per the contract (REQ-API-03/04),
		// e.g. "Intel(R) Xeon(R) ...". cpu_vender_id ("GenuineIntel") has no
		// output field, so it is intentionally not stored.
		m.CpuInfo = si.CpuBrand
		m.CpuModel = si.CpuBrand
	}
	if sr.IpInfo != nil {
		m.Region = sr.IpInfo.Country // server-injected ISO country code (REQ-API-02)
	}
	return m
}

// ── small mappers ────────────────────────────────────────────────────────────

func nowISO() string { return time.Now().UTC().Format(time.RFC3339) }

func nowUnix() int64 { return time.Now().Unix() }

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func formatLoad(l1, l5, l15 float64) string {
	return strconv.FormatFloat(l1, 'f', -1, 64) + " " +
		strconv.FormatFloat(l5, 'f', -1, 64) + " " +
		strconv.FormatFloat(l15, 'f', -1, 64)
}

// pingPtr maps a wire ping value (ms) to a nullable int: -1/negative -> NULL.
func pingPtr(v float64) *int64 {
	if v < 0 {
		return nil
	}
	n := int64(v + 0.5)
	return &n
}

// lossPtr maps a wire loss value (%) to a nullable float. Out-of-range values
// (-1/negative sentinel, or >100% anomalies) become NULL (03-report-protocol
// §2.6.3: "clamp or mark NULL").
func lossPtr(v float64) *float64 {
	if v < 0 || v > 100 {
		return nil
	}
	return &v
}

// gpuPtr maps an optional gpu value to a nullable float: nil/negative -> NULL.
func gpuPtr(p *float64) *float64 {
	if p == nil || *p < 0 {
		return nil
	}
	v := *p
	return &v
}

func disksJSON(disks []models.DiskInfo) string {
	if len(disks) == 0 {
		return "[]"
	}
	b, err := json.Marshal(disks)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// marshalSnapshot JSON-encodes an optional struct pointer, returning "" for nil
// (avoids the typed-nil-interface pitfall and the literal "null" string).
func marshalSnapshot[T any](p *T) string {
	if p == nil {
		return ""
	}
	b, err := json.Marshal(p)
	if err != nil {
		return ""
	}
	return string(b)
}

func orDefaultInt(v, def int) int {
	if v > 0 {
		return v
	}
	return def
}

func orDefaultStr(v, def string) string {
	if v != "" {
		return v
	}
	return def
}
