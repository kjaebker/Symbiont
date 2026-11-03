package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	// Required
	ApexURL  string
	ApexUser string
	ApexPass string

	// Database paths
	DBPath     string
	SQLitePath string

	// Poller
	PollInterval time.Duration

	// API server
	APIPort string
	Token   string

	// Logging
	LogLevel string

	// Retention
	RetentionDays int

	// Notifications
	NtfyURL string
	// NotifyAllowedHosts lists host:port pairs that notification targets may
	// use even though they are private or loopback addresses (and over plain
	// http), e.g. a notification server on the same machine. Empty by default.
	NotifyAllowedHosts []string

	// Frontend
	FrontendPath string

	// System
	HeartbeatPath string
	BackupDir     string
}

// Load reads configuration from the environment, loading .env if present.
// Fatal if required fields are missing.
func Load() *Config {
	// Load .env file if it exists — ignore error (file is optional in production).
	_ = godotenv.Load()

	cfg := &Config{
		ApexURL:  os.Getenv("SYMBIONT_APEX_URL"),
		ApexUser: os.Getenv("SYMBIONT_APEX_USER"),
		ApexPass: os.Getenv("SYMBIONT_APEX_PASS"),

		DBPath:     envOrDefault("SYMBIONT_DB_PATH", "telemetry.db"),
		SQLitePath: envOrDefault("SYMBIONT_SQLITE_PATH", "app.db"),
		APIPort:    envOrDefault("SYMBIONT_API_PORT", "8420"),
		LogLevel:   envOrDefault("SYMBIONT_LOG_LEVEL", "info"),
		Token:      os.Getenv("SYMBIONT_TOKEN"),
		FrontendPath: envOrDefault("SYMBIONT_FRONTEND_PATH", "./frontend/dist"),
		NtfyURL:       os.Getenv("SYMBIONT_NTFY_URL"),
		NotifyAllowedHosts: splitList(os.Getenv("SYMBIONT_NOTIFY_ALLOWED_HOSTS")),
		HeartbeatPath: os.Getenv("SYMBIONT_HEARTBEAT_PATH"),
		BackupDir:     envOrDefault("SYMBIONT_BACKUP_DIR", "./backups"),
	}

	// Parse poll interval duration.
	pollStr := envOrDefault("SYMBIONT_POLL_INTERVAL", "10s")
	d, err := time.ParseDuration(pollStr)
	if err != nil {
		log.Fatalf("config: invalid SYMBIONT_POLL_INTERVAL %q: %v", pollStr, err)
	}
	cfg.PollInterval = d

	// Parse retention days.
	retStr := envOrDefault("SYMBIONT_RETENTION_DAYS", "365")
	ret, err := strconv.Atoi(retStr)
	if err != nil {
		log.Fatalf("config: invalid SYMBIONT_RETENTION_DAYS %q: %v", retStr, err)
	}
	cfg.RetentionDays = ret

	// Validate required fields.
	if cfg.ApexURL == "" {
		log.Fatal("config: SYMBIONT_APEX_URL is required")
	}
	if cfg.ApexUser == "" {
		log.Fatal("config: SYMBIONT_APEX_USER is required")
	}
	if cfg.ApexPass == "" {
		log.Fatal("config: SYMBIONT_APEX_PASS is required")
	}

	return cfg
}

// splitList parses a comma-separated list, trimming spaces and dropping empties.
func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.ToLower(strings.TrimSpace(p)); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
