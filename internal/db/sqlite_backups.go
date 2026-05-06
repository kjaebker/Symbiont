package db

import (
	"context"
	"fmt"
)

// --- Backup Jobs ---

// InsertBackupJob inserts a new backup job record and returns its ID.
func (s *SQLiteDB) InsertBackupJob(ctx context.Context, job BackupJob) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO backup_jobs (status, path, size_bytes, error) VALUES (?, ?, ?, ?)`,
		job.Status, job.Path, job.SizeBytes, job.Error,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting backup job: %w", err)
	}
	return res.LastInsertId()
}

// UpdateBackupJob updates an existing backup job's status and error.
func (s *SQLiteDB) UpdateBackupJob(ctx context.Context, id int64, status string, errMsg string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE backup_jobs SET status = ?, error = ? WHERE id = ?",
		status, errMsg, id,
	)
	if err != nil {
		return fmt.Errorf("updating backup job %d: %w", id, err)
	}
	return nil
}

// ListBackupJobs returns recent backup jobs.
func (s *SQLiteDB) ListBackupJobs(ctx context.Context, limit int) ([]BackupJob, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, ts, status, path, size_bytes, error FROM backup_jobs ORDER BY ts DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("listing backup jobs: %w", err)
	}
	defer rows.Close()

	var jobs []BackupJob
	for rows.Next() {
		var j BackupJob
		if err := rows.Scan(&j.ID, &j.TS, &j.Status, &j.Path, &j.SizeBytes, &j.Error); err != nil {
			return nil, fmt.Errorf("scanning backup job: %w", err)
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}
