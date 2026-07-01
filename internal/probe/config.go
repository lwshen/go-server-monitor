// Package probe implements the monitoring agent: config, collection, network
// quality probing, buffering and upload. P0 stubs the collection/upload; the ring
// buffer is real.
package probe

import (
	"os"
	"strconv"
	"time"
)

// ProbeTarget is one network-quality probe destination (ct/cu/cm/bd).
type ProbeTarget struct {
	Name string // "ct", "cu", "cm", "bd"
	Host string // IP or hostname
}

// Config holds probe runtime configuration (REQ-PROBE-05 / REQ-PROBE-11).
type Config struct {
	ServerURL       string        // SERVER_URL — server base URL (required)
	ServerID        string        // SERVER_ID — this server's UUID (from admin add; required)
	APISecret       string        // API_SECRET / SECRET — shared upload secret (required)
	CollectInterval time.Duration // COLLECT_INTERVAL — sample period (default 5s)
	ReportInterval  time.Duration // REPORT_INTERVAL — upload period (default 60s)
	BufferSize      int           // BUFFER_SIZE — ring buffer capacity (default 300)
	Targets         []ProbeTarget // network-quality probe targets
}

// DefaultTargets are the built-in three-network + Baidu probe hosts (REQ-PROBE-11).
func DefaultTargets() []ProbeTarget {
	return []ProbeTarget{
		{Name: "ct", Host: "219.141.140.10"}, // China Telecom
		{Name: "cu", Host: "210.22.97.1"},    // China Unicom
		{Name: "cm", Host: "211.139.170.1"},  // China Mobile
		{Name: "bd", Host: "119.75.217.56"},  // Baidu
	}
}

// Load reads probe configuration from the environment with defaults.
//
// TODO(P3): also support a config.yaml (server.url/secret, probe.* , targets[]).
func Load() *Config {
	return &Config{
		ServerURL:       env("SERVER_URL", ""),
		ServerID:        firstNonEmpty(os.Getenv("SERVER_ID"), os.Getenv("ID")),
		APISecret:       firstNonEmpty(os.Getenv("API_SECRET"), os.Getenv("SECRET")),
		CollectInterval: envDuration("COLLECT_INTERVAL", 5*time.Second),
		ReportInterval:  envDuration("REPORT_INTERVAL", 60*time.Second),
		BufferSize:      envInt("BUFFER_SIZE", 300),
		Targets:         DefaultTargets(),
	}
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func envInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// envDuration reads an integer number of seconds from the env, or def.
func envDuration(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return time.Duration(n) * time.Second
		}
	}
	return def
}
