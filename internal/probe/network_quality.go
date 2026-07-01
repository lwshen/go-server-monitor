package probe

import (
	"net"
	"sync"
	"time"
)

// probeAttempts is how many connects each probe makes to estimate loss.
const probeAttempts = 4

// probePort is dialed for the TCP-connect latency measurement.
const probePort = "80"

// ProbePing measures latency (ms) and loss (%) to host (REQ-PROBE-07).
//
// It uses a TCP connect (no raw-socket / root privilege needed, and cross-platform)
// as the ICMP fallback the spec permits: it dials probePort probeAttempts times,
// averaging the RTT of successful connects. loss = failed/total × 100. When every
// attempt fails the host is treated as unmeasured and (-1, -1, err) is returned so
// the values store as SQL NULL (CONVENTIONS §2).
func ProbePing(host string, timeout time.Duration) (latency int, loss int, err error) {
	var success int
	var totalRTT time.Duration
	var lastErr error

	for i := 0; i < probeAttempts; i++ {
		start := time.Now()
		conn, e := net.DialTimeout("tcp", net.JoinHostPort(host, probePort), timeout)
		if e != nil {
			lastErr = e
			continue
		}
		totalRTT += time.Since(start)
		_ = conn.Close()
		success++
	}

	loss = (probeAttempts - success) * 100 / probeAttempts
	if success == 0 {
		return -1, -1, lastErr
	}
	return int((totalRTT / time.Duration(success)).Milliseconds()), loss, nil
}

// NetMonitor periodically probes the three-network + Baidu targets and caches the
// latest result per target for the collector to merge into samples.
type NetMonitor struct {
	targets []ProbeTarget
	timeout time.Duration

	mu      sync.RWMutex
	results map[string]netResult // keyed by target name (ct/cu/cm/bd)
}

type netResult struct{ ping, loss int }

// NewNetMonitor seeds every target as unmeasured (-1) until the first probe runs.
func NewNetMonitor(targets []ProbeTarget) *NetMonitor {
	m := &NetMonitor{
		targets: targets,
		timeout: 3 * time.Second,
		results: make(map[string]netResult, len(targets)),
	}
	for _, t := range targets {
		m.results[t.Name] = netResult{ping: -1, loss: -1}
	}
	return m
}

// Probe measures all targets concurrently and stores the results.
func (m *NetMonitor) Probe() {
	var wg sync.WaitGroup
	for _, t := range m.targets {
		wg.Add(1)
		go func(t ProbeTarget) {
			defer wg.Done()
			ping, loss, _ := ProbePing(t.Host, m.timeout)
			m.mu.Lock()
			m.results[t.Name] = netResult{ping: ping, loss: loss}
			m.mu.Unlock()
		}(t)
	}
	wg.Wait()
}

// Get returns the latest (ping ms, loss %) for a target, or (-1, -1) if unknown.
func (m *NetMonitor) Get(name string) (ping, loss int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if r, ok := m.results[name]; ok {
		return r.ping, r.loss
	}
	return -1, -1
}
