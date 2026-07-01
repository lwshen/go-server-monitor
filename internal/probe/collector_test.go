package probe

import (
	"testing"
	"time"
)

// TestCollectSmoke runs the real gopsutil collector on the test host and checks
// the always-true invariants (total memory > 0, sys_info populated, sane units).
// Rate fields (cpu %, network speed) need two samples to be meaningful, so the
// second Collect is what we assert non-negative CPU on.
func TestCollectSmoke(t *testing.T) {
	c := NewCollector(&Config{Targets: DefaultTargets()})

	first, err := c.Collect()
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if first.MemoryTotal <= 0 {
		t.Fatalf("memory_total = %v MiB, want > 0", first.MemoryTotal)
	}
	if first.MemoryUsed < 0 || first.MemoryUsed > first.MemoryTotal {
		t.Fatalf("memory_used = %v, want within [0, %v]", first.MemoryUsed, first.MemoryTotal)
	}
	if first.SysInfo == nil || first.SysInfo.OsName == "" {
		t.Fatalf("sys_info = %+v, want populated", first.SysInfo)
	}
	if first.Frame != "data" || first.LatestTs == 0 {
		t.Fatalf("frame/latest_ts = %q/%d, want data/nonzero", first.Frame, first.LatestTs)
	}

	// Unmeasured network targets must surface the -1 sentinel (nothing probed yet).
	if first.PingCt != -1 {
		t.Fatalf("ping_ct before probe = %v, want -1 (unmeasured)", first.PingCt)
	}

	// A second sample yields a CPU rate; it must be a valid percentage.
	second, err := c.Collect()
	if err != nil {
		t.Fatalf("second Collect: %v", err)
	}
	if second.Cpu < 0 || second.Cpu > 100*float64(max(1, second.SysInfo.CpuNum)) {
		t.Fatalf("cpu = %v, out of range", second.Cpu)
	}
	_ = time.Now
}
