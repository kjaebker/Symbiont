package backup

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/kjaebker/symbiont/internal/db"
)

func TestRunBackup(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a real DuckDB file.
	duckPath := filepath.Join(tmpDir, "test.duckdb")
	duck, err := db.Open(duckPath)
	if err != nil {
		t.Fatalf("opening duckdb: %v", err)
	}
	defer duck.Close()

	// Create a real SQLite file.
	sqlitePath := filepath.Join(tmpDir, "test.sqlite")
	sqlite, err := db.OpenSQLite(sqlitePath)
	if err != nil {
		t.Fatalf("opening sqlite: %v", err)
	}
	defer sqlite.Close()

	backupDir := filepath.Join(tmpDir, "backups")
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	result, err := Run(context.Background(), duck, sqlite, Config{
		BackupDir: backupDir,
		Retain:    3,
	}, logger)
	if err != nil {
		t.Fatalf("backup failed: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("expected status 'success', got %q", result.Status)
	}
	if len(result.Paths) != 2 {
		t.Fatalf("expected 2 backup paths, got %d", len(result.Paths))
	}
	if result.SizeBytes <= 0 {
		t.Error("expected positive size")
	}

	// Verify files exist.
	for _, p := range result.Paths {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("backup file not found: %s", p)
		}
	}
}

func TestPruneOldBackups(t *testing.T) {
	dir := t.TempDir()

	// Create fake backup files for 5 dates.
	dates := []string{"2025-03-16", "2025-03-17", "2025-03-18", "2025-03-19", "2025-03-20"}
	for _, d := range dates {
		os.WriteFile(filepath.Join(dir, "telemetry-"+d+".db"), []byte("data"), 0o644)
		os.WriteFile(filepath.Join(dir, "app-"+d+".db"), []byte("data"), 0o644)
	}

	// Keep only 2.
	err := PruneOldBackups(dir, 2)
	if err != nil {
		t.Fatalf("prune failed: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading dir: %v", err)
	}

	// Should have 4 files (2 dates × 2 files each).
	if len(entries) != 4 {
		t.Errorf("expected 4 files after pruning, got %d", len(entries))
		for _, e := range entries {
			t.Logf("  %s", e.Name())
		}
	}

	// The newest two dates should survive.
	for _, e := range entries {
		name := e.Name()
		if !contains(name, "2025-03-19") && !contains(name, "2025-03-20") {
			t.Errorf("unexpected file survived pruning: %s", name)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && findSubstr(s, substr))
}

func findSubstr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestBackupRecordsJob(t *testing.T) {
	tmpDir := t.TempDir()

	duckPath := filepath.Join(tmpDir, "test.duckdb")
	duck, err := db.Open(duckPath)
	if err != nil {
		t.Fatalf("opening duckdb: %v", err)
	}
	defer duck.Close()

	sqlite, err := db.OpenSQLite(filepath.Join(tmpDir, "test.sqlite"))
	if err != nil {
		t.Fatalf("opening sqlite: %v", err)
	}
	defer sqlite.Close()

	backupDir := filepath.Join(tmpDir, "backups")
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	result, err := Run(context.Background(), duck, sqlite, Config{
		BackupDir: backupDir,
		Retain:    3,
	}, logger)
	if err != nil {
		t.Fatalf("backup failed: %v", err)
	}

	// Record the job manually (as the API handler would).
	ctx := context.Background()
	var path *string
	if len(result.Paths) > 0 {
		p := result.Paths[0]
		path = &p
	}
	_, err = sqlite.InsertBackupJob(ctx, db.BackupJob{
		Status:    "success",
		Path:      path,
		SizeBytes: &result.SizeBytes,
	})
	if err != nil {
		t.Fatalf("inserting backup job: %v", err)
	}

	jobs, err := sqlite.ListBackupJobs(ctx, 10)
	if err != nil {
		t.Fatalf("listing backup jobs: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].Status != "success" {
		t.Errorf("expected status 'success', got %q", jobs[0].Status)
	}
}
