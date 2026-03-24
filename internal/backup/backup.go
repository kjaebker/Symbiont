package backup

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kjaebker/symbiont/internal/db"
)

// Config holds backup configuration.
type Config struct {
	BackupDir string
	Retain    int // Number of backups to keep.
}

// Result describes the outcome of a backup run.
type Result struct {
	Status    string `json:"status"`
	Paths     []string `json:"paths"`
	SizeBytes int64  `json:"size_bytes"`
	Error     string `json:"error,omitempty"`
}

// Run performs a backup of DuckDB and SQLite databases.
func Run(ctx context.Context, duck *db.DuckDB, sqlite *db.SQLiteDB, cfg Config, logger *slog.Logger) (*Result, error) {
	if err := os.MkdirAll(cfg.BackupDir, 0o750); err != nil {
		return nil, fmt.Errorf("creating backup dir: %w", err)
	}

	stamp := time.Now().Format("2006-01-02")

	// Checkpoint DuckDB before copy.
	if _, err := duck.DB().ExecContext(ctx, "CHECKPOINT"); err != nil {
		logger.Warn("duckdb checkpoint failed, proceeding with copy", "err", err)
	}

	// Checkpoint SQLite WAL before copy.
	if _, err := sqlite.DB().ExecContext(ctx, "PRAGMA wal_checkpoint(FULL)"); err != nil {
		logger.Warn("sqlite wal checkpoint failed, proceeding with copy", "err", err)
	}

	var paths []string
	var totalSize int64

	// Copy DuckDB.
	duckSrc := duck.Path()
	if duckSrc != "" {
		duckDst := filepath.Join(cfg.BackupDir, fmt.Sprintf("telemetry-%s.db", stamp))
		size, err := copyFile(duckSrc, duckDst)
		if err != nil {
			return &Result{Status: "failed", Error: fmt.Sprintf("copying duckdb: %v", err)}, err
		}
		paths = append(paths, duckDst)
		totalSize += size
		logger.Info("backed up duckdb", "path", duckDst, "size_bytes", size)
	}

	// Copy SQLite.
	sqliteSrc := sqlite.Path()
	if sqliteSrc != "" && sqliteSrc != ":memory:" {
		sqliteDst := filepath.Join(cfg.BackupDir, fmt.Sprintf("app-%s.db", stamp))
		size, err := copyFile(sqliteSrc, sqliteDst)
		if err != nil {
			return &Result{Status: "failed", Error: fmt.Sprintf("copying sqlite: %v", err)}, err
		}
		paths = append(paths, sqliteDst)
		totalSize += size
		logger.Info("backed up sqlite", "path", sqliteDst, "size_bytes", size)
	}

	// Prune old backups.
	if cfg.Retain > 0 {
		if err := PruneOldBackups(cfg.BackupDir, cfg.Retain); err != nil {
			logger.Warn("backup pruning failed", "err", err)
		}
	}

	return &Result{
		Status:    "success",
		Paths:     paths,
		SizeBytes: totalSize,
	}, nil
}

// PruneOldBackups keeps only the most recent `retain` backup sets.
// A backup set is identified by date stamp in the filename.
func PruneOldBackups(dir string, retain int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading backup dir: %w", err)
	}

	// Collect unique date stamps.
	stamps := make(map[string]bool)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		// Parse date from filenames like "telemetry-2025-03-20.db" or "app-2025-03-20.db".
		name := e.Name()
		if idx := strings.Index(name, "-"); idx >= 0 {
			rest := name[idx+1:]
			if ext := strings.Index(rest, ".db"); ext >= 0 {
				stamp := rest[:ext]
				stamps[stamp] = true
			}
		}
	}

	sorted := make([]string, 0, len(stamps))
	for s := range stamps {
		sorted = append(sorted, s)
	}
	sort.Strings(sorted)

	if len(sorted) <= retain {
		return nil
	}

	// Delete files for oldest stamps.
	toDelete := sorted[:len(sorted)-retain]
	deleteSet := make(map[string]bool, len(toDelete))
	for _, s := range toDelete {
		deleteSet[s] = true
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		for stamp := range deleteSet {
			if strings.Contains(e.Name(), stamp) {
				path := filepath.Join(dir, e.Name())
				if err := os.Remove(path); err != nil {
					return fmt.Errorf("removing old backup %s: %w", path, err)
				}
			}
		}
	}

	return nil
}

func copyFile(src, dst string) (int64, error) {
	in, err := os.Open(src)
	if err != nil {
		return 0, fmt.Errorf("opening %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return 0, fmt.Errorf("creating %s: %w", dst, err)
	}
	defer out.Close()

	n, err := io.Copy(out, in)
	if err != nil {
		return 0, fmt.Errorf("copying %s to %s: %w", src, dst, err)
	}

	if err := out.Sync(); err != nil {
		return 0, fmt.Errorf("syncing %s: %w", dst, err)
	}

	return n, nil
}
