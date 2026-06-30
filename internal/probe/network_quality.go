package probe

import "time"

// ProbePing measures latency (ms) and loss (%) to host (REQ-PROBE-07).
//
// P0 STUB: returns the unmeasured sentinel (-1, -1, nil) per CONVENTIONS.md §2.
//
// TODO(P3): ICMP ping (4 probes), degrading to TCP/HTTP without raw-socket
// privilege; return average RTT ms and loss %, or (-1, -1, err) on failure.
func ProbePing(host string, timeout time.Duration) (latency int, loss int, err error) {
	_ = host
	_ = timeout
	return -1, -1, nil
}
