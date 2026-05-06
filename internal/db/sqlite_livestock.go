package db

import (
	"context"
	"fmt"
	"strings"
)

// --- Livestock ---

// ListLivestock returns livestock items matching the filter, ordered by name.
func (s *SQLiteDB) ListLivestock(ctx context.Context, f LivestockFilter) ([]LivestockItem, error) {
	query := `SELECT id, name, species, type, quantity, status, date_added, notes, image_path, created_at, updated_at
		FROM livestock`

	var conditions []string
	var args []any

	if f.Type != "" {
		conditions = append(conditions, "type = ?")
		args = append(args, f.Type)
	}
	if f.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, f.Status)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY name"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing livestock: %w", err)
	}
	defer rows.Close()

	var items []LivestockItem
	for rows.Next() {
		var item LivestockItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Species, &item.Type, &item.Quantity, &item.Status, &item.DateAdded, &item.Notes, &item.ImagePath, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning livestock item: %w", err)
		}
		items = append(items, item)
	}
	if items == nil {
		items = []LivestockItem{}
	}
	return items, rows.Err()
}

// ListLivestockSpecies returns distinct non-null species values, ordered alphabetically.
func (s *SQLiteDB) ListLivestockSpecies(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT DISTINCT species FROM livestock WHERE species IS NOT NULL ORDER BY species",
	)
	if err != nil {
		return nil, fmt.Errorf("listing livestock species: %w", err)
	}
	defer rows.Close()

	var species []string
	for rows.Next() {
		var sp string
		if err := rows.Scan(&sp); err != nil {
			return nil, fmt.Errorf("scanning species: %w", err)
		}
		species = append(species, sp)
	}
	if species == nil {
		species = []string{}
	}
	return species, rows.Err()
}

// GetLivestockItem returns a single livestock item by ID.
func (s *SQLiteDB) GetLivestockItem(ctx context.Context, id int64) (*LivestockItem, error) {
	var item LivestockItem
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, species, type, quantity, status, date_added, notes, image_path, created_at, updated_at
		FROM livestock WHERE id = ?`, id,
	).Scan(&item.ID, &item.Name, &item.Species, &item.Type, &item.Quantity, &item.Status, &item.DateAdded, &item.Notes, &item.ImagePath, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// InsertLivestockItem inserts a new livestock item and returns its ID.
func (s *SQLiteDB) InsertLivestockItem(ctx context.Context, item LivestockItem) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO livestock (name, species, type, quantity, status, date_added, notes, image_path)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		item.Name, item.Species, item.Type, item.Quantity, item.Status, item.DateAdded, item.Notes, item.ImagePath,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting livestock item: %w", err)
	}
	return res.LastInsertId()
}

// UpdateLivestockItem updates an existing livestock item.
func (s *SQLiteDB) UpdateLivestockItem(ctx context.Context, id int64, item LivestockItem) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE livestock SET name = ?, species = ?, type = ?, quantity = ?, status = ?, date_added = ?, notes = ?, image_path = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		item.Name, item.Species, item.Type, item.Quantity, item.Status, item.DateAdded, item.Notes, item.ImagePath, id,
	)
	if err != nil {
		return fmt.Errorf("updating livestock item %d: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("livestock item %d not found", id)
	}
	return nil
}

// SetLivestockImagePath updates only the image_path for a livestock item.
// Used by the image edit and reset flows to avoid clobbering metadata.
func (s *SQLiteDB) SetLivestockImagePath(ctx context.Context, id int64, imagePath *string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE livestock SET image_path = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		imagePath, id,
	)
	if err != nil {
		return fmt.Errorf("setting livestock image path %d: %w", id, err)
	}
	return nil
}

// DeleteLivestockItem removes a livestock item by ID.
func (s *SQLiteDB) DeleteLivestockItem(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM livestock WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting livestock item %d: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("livestock item %d not found", id)
	}
	return nil
}

// ListLivestockObservations returns all observations for a livestock item, newest first.
func (s *SQLiteDB) ListLivestockObservations(ctx context.Context, livestockID int64) ([]LivestockObservation, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, livestock_id, ts, status, note, image_path FROM livestock_observations WHERE livestock_id = ? ORDER BY ts DESC",
		livestockID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing livestock observations: %w", err)
	}
	defer rows.Close()

	var obs []LivestockObservation
	for rows.Next() {
		var o LivestockObservation
		if err := rows.Scan(&o.ID, &o.LivestockID, &o.TS, &o.Status, &o.Note, &o.ImagePath); err != nil {
			return nil, fmt.Errorf("scanning livestock observation: %w", err)
		}
		obs = append(obs, o)
	}
	if obs == nil {
		obs = []LivestockObservation{}
	}
	return obs, rows.Err()
}

// GetLivestockObservation returns a single observation by ID.
func (s *SQLiteDB) GetLivestockObservation(ctx context.Context, id int64) (*LivestockObservation, error) {
	var o LivestockObservation
	err := s.db.QueryRowContext(ctx,
		"SELECT id, livestock_id, ts, status, note, image_path FROM livestock_observations WHERE id = ?",
		id,
	).Scan(&o.ID, &o.LivestockID, &o.TS, &o.Status, &o.Note, &o.ImagePath)
	if err != nil {
		return nil, fmt.Errorf("getting livestock observation: %w", err)
	}
	return &o, nil
}

// UpdateLivestockObservationImagePath sets or clears the image_path for an observation.
func (s *SQLiteDB) UpdateLivestockObservationImagePath(ctx context.Context, id int64, imagePath *string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE livestock_observations SET image_path = ? WHERE id = ?",
		imagePath, id,
	)
	if err != nil {
		return fmt.Errorf("updating livestock observation image: %w", err)
	}
	return nil
}

// ListAllImagesWithIDs returns every non-null image alongside the IDs needed
// to compute a deterministic filename. Used by the image reprocess migrator.
func (s *SQLiteDB) ListAllImagesWithIDs(ctx context.Context) ([]ImageRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT 'livestock', id, id, image_path FROM livestock WHERE image_path IS NOT NULL
		UNION ALL
		SELECT 'observation', id, livestock_id, image_path FROM livestock_observations WHERE image_path IS NOT NULL
	`)
	if err != nil {
		return nil, fmt.Errorf("listing images with ids: %w", err)
	}
	defer rows.Close()
	var records []ImageRecord
	for rows.Next() {
		var r ImageRecord
		if err := rows.Scan(&r.Kind, &r.ID, &r.LivestockID, &r.ImagePath); err != nil {
			return nil, fmt.Errorf("scanning image record: %w", err)
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// ListAllImagePaths returns every non-null image_path stored across the
// livestock and livestock_observations tables. Used by the image reprocessor.
func (s *SQLiteDB) ListAllImagePaths(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT image_path FROM livestock WHERE image_path IS NOT NULL
		UNION ALL
		SELECT image_path FROM livestock_observations WHERE image_path IS NOT NULL
	`)
	if err != nil {
		return nil, fmt.Errorf("listing image paths: %w", err)
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, fmt.Errorf("scanning image path: %w", err)
		}
		paths = append(paths, p)
	}
	return paths, rows.Err()
}

// InsertLivestockObservation inserts a new observation and returns its ID.
func (s *SQLiteDB) InsertLivestockObservation(ctx context.Context, o LivestockObservation) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		"INSERT INTO livestock_observations (livestock_id, status, note) VALUES (?, ?, ?)",
		o.LivestockID, o.Status, o.Note,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting livestock observation: %w", err)
	}
	return res.LastInsertId()
}
