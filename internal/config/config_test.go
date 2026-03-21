package config

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestDefaults(t *testing.T) {
	// Set only required fields; verify optional fields get defaults.
	t.Setenv("SYMBIONT_APEX_URL", "http://192.168.1.1")
	t.Setenv("SYMBIONT_APEX_USER", "admin")
	t.Setenv("SYMBIONT_APEX_PASS", "secret")

	// Clear optional fields to ensure defaults apply.
	clearOptional(t)

	cfg := Load()

	if cfg.DBPath != "telemetry.db" {
		t.Errorf("DBPath default: got %q, want %q", cfg.DBPath, "telemetry.db")
	}
	if cfg.SQLitePath != "app.db" {
		t.Errorf("SQLitePath default: got %q, want %q", cfg.SQLitePath, "app.db")
	}
	if cfg.APIPort != "8420" {
		t.Errorf("APIPort default: got %q, want %q", cfg.APIPort, "8420")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel default: got %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.PollInterval != 10*time.Second {
		t.Errorf("PollInterval default: got %v, want %v", cfg.PollInterval, 10*time.Second)
	}
	if cfg.RetentionDays != 365 {
		t.Errorf("RetentionDays default: got %d, want %d", cfg.RetentionDays, 365)
	}
}

func TestRequiredFieldsLoaded(t *testing.T) {
	t.Setenv("SYMBIONT_APEX_URL", "http://10.0.0.50")
	t.Setenv("SYMBIONT_APEX_USER", "operator")
	t.Setenv("SYMBIONT_APEX_PASS", "hunter2")
	clearOptional(t)

	cfg := Load()

	if cfg.ApexURL != "http://10.0.0.50" {
		t.Errorf("ApexURL: got %q", cfg.ApexURL)
	}
	if cfg.ApexUser != "operator" {
		t.Errorf("ApexUser: got %q", cfg.ApexUser)
	}
	if cfg.ApexPass != "hunter2" {
		t.Errorf("ApexPass: got %q", cfg.ApexPass)
	}
}

func TestOptionalOverrides(t *testing.T) {
	t.Setenv("SYMBIONT_APEX_URL", "http://192.168.1.1")
	t.Setenv("SYMBIONT_APEX_USER", "admin")
	t.Setenv("SYMBIONT_APEX_PASS", "secret")
	t.Setenv("SYMBIONT_DB_PATH", "/data/telemetry.db")
	t.Setenv("SYMBIONT_SQLITE_PATH", "/data/app.db")
	t.Setenv("SYMBIONT_API_PORT", "9000")
	t.Setenv("SYMBIONT_LOG_LEVEL", "debug")
	t.Setenv("SYMBIONT_POLL_INTERVAL", "30s")
	t.Setenv("SYMBIONT_RETENTION_DAYS", "90")

	cfg := Load()

	if cfg.DBPath != "/data/telemetry.db" {
		t.Errorf("DBPath: got %q", cfg.DBPath)
	}
	if cfg.SQLitePath != "/data/app.db" {
		t.Errorf("SQLitePath: got %q", cfg.SQLitePath)
	}
	if cfg.APIPort != "9000" {
		t.Errorf("APIPort: got %q", cfg.APIPort)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel: got %q", cfg.LogLevel)
	}
	if cfg.PollInterval != 30*time.Second {
		t.Errorf("PollInterval: got %v", cfg.PollInterval)
	}
	if cfg.RetentionDays != 90 {
		t.Errorf("RetentionDays: got %d", cfg.RetentionDays)
	}
}

func TestPollIntervalParsing(t *testing.T) {
	cases := []struct {
		input string
		want  time.Duration
	}{
		{"10s", 10 * time.Second},
		{"1m", time.Minute},
		{"30s", 30 * time.Second},
		{"5m", 5 * time.Minute},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			clearOptional(t)
			t.Setenv("SYMBIONT_APEX_URL", "http://192.168.1.1")
			t.Setenv("SYMBIONT_APEX_USER", "admin")
			t.Setenv("SYMBIONT_APEX_PASS", "secret")
			t.Setenv("SYMBIONT_POLL_INTERVAL", tc.input)

			cfg := Load()
			if cfg.PollInterval != tc.want {
				t.Errorf("PollInterval(%q): got %v, want %v", tc.input, cfg.PollInterval, tc.want)
			}
		})
	}
}

// TestMissingRequiredFieldFatal verifies that missing required fields cause process exit.
// Uses subprocess execution to safely test log.Fatal behavior.
func TestMissingRequiredFieldFatal(t *testing.T) {
	cases := []struct {
		name    string
		missing string
	}{
		{"missing apex url", "SYMBIONT_APEX_URL"},
		{"missing apex user", "SYMBIONT_APEX_USER"},
		{"missing apex pass", "SYMBIONT_APEX_PASS"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=TestFatalHelper")
			cmd.Env = append(os.Environ(),
				"SYMBIONT_FATAL_HELPER=1",
				"SYMBIONT_FATAL_MISSING="+tc.missing,
			)
			err := cmd.Run()
			if err == nil {
				t.Errorf("expected non-zero exit for missing %s, got success", tc.missing)
			}
		})
	}
}

// TestFatalHelper is a helper test invoked as a subprocess by TestMissingRequiredFieldFatal.
// It is not run directly.
func TestFatalHelper(t *testing.T) {
	if os.Getenv("SYMBIONT_FATAL_HELPER") != "1" {
		return
	}

	// Set all required fields, then unset the one we're testing.
	os.Setenv("SYMBIONT_APEX_URL", "http://192.168.1.1")
	os.Setenv("SYMBIONT_APEX_USER", "admin")
	os.Setenv("SYMBIONT_APEX_PASS", "secret")

	missing := os.Getenv("SYMBIONT_FATAL_MISSING")
	os.Unsetenv(missing)
	clearOptional(t)

	Load() // must call os.Exit(1)
}

// clearOptional unsets all optional env vars so tests start from defaults.
func clearOptional(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"SYMBIONT_DB_PATH",
		"SYMBIONT_SQLITE_PATH",
		"SYMBIONT_API_PORT",
		"SYMBIONT_LOG_LEVEL",
		"SYMBIONT_POLL_INTERVAL",
		"SYMBIONT_RETENTION_DAYS",
		"SYMBIONT_TOKEN",
		"SYMBIONT_NTFY_URL",
		"SYMBIONT_BACKUP_DIR",
	} {
		t.Setenv(key, "")
	}
}
