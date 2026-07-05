// Package config loads the server's DEPLOY-time configuration from environment
// variables (a .env file is auto-loaded via godotenv in main), falling back to
// built-in defaults. There is intentionally no config file — env is the single
// source for deploy config (works identically for .env, docker-compose
// `environment:`, and systemd `EnvironmentFile=`).
//
// Runtime / operational settings (site title, theme, alert channels, retention,
// offline factor, …) instead live in the DB settings table and are edited via the
// admin API; the secret- and operational-seed values below are written into that
// table on first start (see service.BootstrapAdmin / BootstrapSettings), after
// which the DB is authoritative.
package config

import (
	"os"
	"strconv"
)

// Config holds all runtime configuration for the server process.
//
// Secret-class values (APISecret, AdminPassword, JWTSecret) are bootstrapped from
// the environment on first start and then persisted to the settings table
// (REQ-RES-01). For the skeleton they are simply surfaced here.
type Config struct {
	APISecret     string // API_SECRET — probe upload shared secret ("" disables /report)
	AdminUsername string // ADMIN_USERNAME — admin login username (default "admin")
	AdminPassword string // ADMIN_PASSWORD — plaintext bootstrap password, bcrypt-hashed on first start
	JWTSecret     string // JWT_SECRET — JWT signing key (derived from ADMIN_PASSWORD if empty)

	// DatabaseURL selects the backend by scheme (store.Open):
	//   sqlite:./data/metrics.db | file:./data/metrics.db   -> embedded SQLite
	//   libsql://<db>.turso.io | https://<db>.turso.io       -> Turso/libSQL (remote)
	//   postgres://user:pass@host:5432/db                    -> PostgreSQL (extension point)
	// When empty, DBPath is used as an embedded SQLite file (back-compat default).
	DatabaseURL       string // DATABASE_URL
	DatabaseAuthToken string // DATABASE_AUTH_TOKEN — Turso/libSQL auth token (if not in the URL)
	DBPath            string // DB_PATH — SQLite file path, used only when DATABASE_URL is empty

	ListenAddr          string // LISTEN_ADDR — HTTP listen address
	CORSOrigins         string // CORS_ORIGINS — comma-separated allowed origins ("" = same-origin)
	ReportRetentionDays int    // REPORT_RETENTION_DAYS — history retention window
	OfflineFactor       int    // OFFLINE_FACTOR — offline threshold = factor × report_interval
	LogLevel            string // LOG_LEVEL — debug/info/warn/error
}

// Load reads configuration from the environment, applying built-in defaults.
func Load() (*Config, error) {
	cfg := &Config{
		APISecret:           env("API_SECRET", ""),
		AdminUsername:       env("ADMIN_USERNAME", "admin"),
		AdminPassword:       env("ADMIN_PASSWORD", ""),
		JWTSecret:           env("JWT_SECRET", ""),
		DatabaseURL:         env("DATABASE_URL", ""),
		DatabaseAuthToken:   env("DATABASE_AUTH_TOKEN", ""),
		DBPath:              env("DB_PATH", "./data/metrics.db"),
		ListenAddr:          env("LISTEN_ADDR", ":8080"),
		CORSOrigins:         env("CORS_ORIGINS", ""),
		ReportRetentionDays: envInt("REPORT_RETENTION_DAYS", 180),
		OfflineFactor:       envInt("OFFLINE_FACTOR", 5),
		LogLevel:            env("LOG_LEVEL", "info"),
	}

	// JWT_SECRET defaults to being derived from ADMIN_PASSWORD when unset.
	// TODO(P6): derive a stable key (e.g. HKDF over ADMIN_PASSWORD) and persist to settings.
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = cfg.AdminPassword
	}

	return cfg, nil
}

// env returns the environment variable named key, or def if unset/empty.
func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

// envInt returns the integer environment variable named key, or def on miss/parse error.
func envInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
