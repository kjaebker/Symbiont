package api

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kjaebker/symbiont/internal/db"
)

// validDeviceTypes is the set of known device types enforced at the app layer.
var validDeviceTypes = map[string]bool{
	"heater":    true,
	"pump":      true,
	"wavemaker": true,
	"light":     true,
	"skimmer":   true,
	"reactor":   true,
	"doser":     true,
	"ato":       true,
	"chiller":   true,
	"fan":       true,
	"other":     true,
}

const maxImageSize = 5 << 20 // 5MB

func (s *Server) HandleDeviceList(w http.ResponseWriter, r *http.Request) {
	devices, err := s.sqlite.ListDevices(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch devices", "db_error")
		return
	}
	if devices == nil {
		devices = []db.Device{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"devices": devices})
}

func (s *Server) HandleDeviceGet(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(pathValue(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid device id", "invalid_param")
		return
	}

	device, err := s.sqlite.GetDevice(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "device not found", "not_found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch device", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, device)
}

func (s *Server) HandleDeviceCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var body struct {
		Name        string   `json:"name"`
		DeviceType  *string  `json:"device_type"`
		Description *string  `json:"description"`
		Brand       *string  `json:"brand"`
		Model       *string  `json:"model"`
		Notes       *string  `json:"notes"`
		OutletID    *string  `json:"outlet_id"`
		ProbeNames  []string `json:"probe_names"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "invalid_body")
		return
	}

	if body.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required", "missing_field")
		return
	}
	if body.DeviceType != nil && *body.DeviceType != "" && !validDeviceTypes[*body.DeviceType] {
		writeError(w, http.StatusBadRequest, "invalid device_type", "invalid_field")
		return
	}

	d := db.Device{
		Name:        body.Name,
		DeviceType:  body.DeviceType,
		Description: body.Description,
		Brand:       body.Brand,
		Model:       body.Model,
		Notes:       body.Notes,
		OutletID:    body.OutletID,
	}

	id, err := s.sqlite.InsertDevice(ctx, d)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			writeError(w, http.StatusConflict, "outlet already linked to another device", "duplicate_outlet")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create device", "db_error")
		return
	}

	// Link probes if provided.
	if len(body.ProbeNames) > 0 {
		if err := s.sqlite.SetDeviceProbes(ctx, id, body.ProbeNames); err != nil {
			s.logger.Error("failed to link probes to device", "err", err, "device_id", id)
		}
	}

	// Sync display names.
	if err := s.sqlite.SyncDeviceDisplayNames(ctx, id, body.Name); err != nil {
		s.logger.Error("failed to sync device display names", "err", err, "device_id", id)
	}

	// Auto-add to dashboard.
	refID := fmt.Sprintf("%d", id)
	if _, err := s.sqlite.AddDashboardItem(ctx, db.DashboardItem{ItemType: "device", ReferenceID: &refID}); err != nil {
		s.logger.Error("failed to add device to dashboard", "err", err, "device_id", id)
	}

	device, err := s.sqlite.GetDevice(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch created device", "db_error")
		return
	}
	writeJSON(w, http.StatusCreated, device)
}

func (s *Server) HandleDeviceUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := strconv.ParseInt(pathValue(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid device id", "invalid_param")
		return
	}

	existing, err := s.sqlite.GetDevice(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "device not found", "not_found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch device", "db_error")
		return
	}

	var body struct {
		Name        string   `json:"name"`
		DeviceType  *string  `json:"device_type"`
		Description *string  `json:"description"`
		Brand       *string  `json:"brand"`
		Model       *string  `json:"model"`
		Notes       *string  `json:"notes"`
		OutletID    *string  `json:"outlet_id"`
		ProbeNames  []string `json:"probe_names"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "invalid_body")
		return
	}

	if body.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required", "missing_field")
		return
	}
	if body.DeviceType != nil && *body.DeviceType != "" && !validDeviceTypes[*body.DeviceType] {
		writeError(w, http.StatusBadRequest, "invalid device_type", "invalid_field")
		return
	}

	d := db.Device{
		Name:        body.Name,
		DeviceType:  body.DeviceType,
		Description: body.Description,
		Brand:       body.Brand,
		Model:       body.Model,
		Notes:       body.Notes,
		OutletID:    body.OutletID,
		ImagePath:   existing.ImagePath,
	}

	if err := s.sqlite.UpdateDevice(ctx, id, d); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			writeError(w, http.StatusConflict, "outlet already linked to another device", "duplicate_outlet")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update device", "db_error")
		return
	}

	// If name changed, sync display names.
	if body.Name != existing.Name {
		if err := s.sqlite.SyncDeviceDisplayNames(ctx, id, body.Name); err != nil {
			s.logger.Error("failed to sync device display names", "err", err, "device_id", id)
		}
	}

	device, err := s.sqlite.GetDevice(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch updated device", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, device)
}

func (s *Server) HandleDeviceDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(pathValue(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid device id", "invalid_param")
		return
	}

	// Delete image file if present.
	device, err := s.sqlite.GetDevice(r.Context(), id)
	if err == nil && device.ImagePath != nil {
		imgPath := s.dataFilePath(*device.ImagePath)
		os.Remove(imgPath)
	}

	if err := s.sqlite.DeleteDevice(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "device not found", "not_found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete device", "db_error")
		return
	}

	// Remove from dashboard.
	refID := fmt.Sprintf("%d", id)
	if err := s.sqlite.RemoveDashboardItemByRef(r.Context(), "device", refID); err != nil {
		s.logger.Error("failed to remove device from dashboard", "err", err, "device_id", id)
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) HandleDeviceSetProbes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := strconv.ParseInt(pathValue(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid device id", "invalid_param")
		return
	}

	// Verify device exists.
	device, err := s.sqlite.GetDevice(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "device not found", "not_found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch device", "db_error")
		return
	}

	var body struct {
		ProbeNames []string `json:"probe_names"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "invalid_body")
		return
	}

	if err := s.sqlite.SetDeviceProbes(ctx, id, body.ProbeNames); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to set device probes", "db_error")
		return
	}

	// Sync display names for the new probe set.
	if err := s.sqlite.SyncDeviceDisplayNames(ctx, id, device.Name); err != nil {
		s.logger.Error("failed to sync device display names", "err", err, "device_id", id)
	}

	updated, err := s.sqlite.GetDevice(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch device", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) HandleDeviceSuggestions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Load current probes from DuckDB.
	readings, err := s.duck.CurrentProbeReadings(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch probes", "db_error")
		return
	}

	// Load current outlets from DuckDB.
	outlets, err := s.duck.CurrentOutletStates(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch outlets", "db_error")
		return
	}

	// Load existing devices to exclude already-linked outlets.
	devices, err := s.sqlite.ListDevices(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch devices", "db_error")
		return
	}
	linkedOutlets := make(map[string]bool, len(devices))
	for _, d := range devices {
		if d.OutletID != nil {
			linkedOutlets[*d.OutletID] = true
		}
	}

	// Build maps for correlation.
	type suggestion struct {
		OutletName    string   `json:"outlet_name"`
		OutletID      string   `json:"outlet_id"`
		ProbeNames    []string `json:"probe_names"`
		SuggestedName string   `json:"suggested_name"`
	}

	// Build outlet name → DID map.
	outletNameToDID := make(map[string]string, len(outlets))
	outletNameMap := make(map[string]string, len(outlets))
	for _, o := range outlets {
		outletNameToDID[o.Name] = o.DID
		outletNameMap[o.DID] = o.Name
	}

	// Find W/A probes and match to outlets.
	var suggestions []suggestion
	wattsProbes := make(map[string]string)  // base → probe name
	ampsProbes := make(map[string]string)   // base → probe name
	for _, rd := range readings {
		if strings.HasSuffix(rd.Name, "W") && rd.Type == "pwr" {
			wattsProbes[strings.TrimSuffix(rd.Name, "W")] = rd.Name
		}
		if strings.HasSuffix(rd.Name, "A") && rd.Type == "Amps" {
			ampsProbes[strings.TrimSuffix(rd.Name, "A")] = rd.Name
		}
	}

	// Match outlets with their power probes.
	for _, o := range outlets {
		if o.Type != "outlet" && o.Type != "virtual" {
			continue
		}
		if linkedOutlets[o.DID] {
			continue
		}

		var probes []string
		if w, ok := wattsProbes[o.Name]; ok {
			probes = append(probes, w)
		}
		if a, ok := ampsProbes[o.Name]; ok {
			probes = append(probes, a)
		}

		// Only suggest outlets that have at least one power probe.
		if len(probes) == 0 {
			continue
		}

		suggestions = append(suggestions, suggestion{
			OutletName:    o.Name,
			OutletID:      o.DID,
			ProbeNames:    probes,
			SuggestedName: splitCamelCase(o.Name),
		})
	}

	if suggestions == nil {
		suggestions = []suggestion{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"suggestions": suggestions})
}

// HandleDeviceImageUpload handles multipart image upload for a device.
func (s *Server) HandleDeviceImageUpload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := strconv.ParseInt(pathValue(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid device id", "invalid_param")
		return
	}

	device, err := s.sqlite.GetDevice(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "device not found", "not_found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch device", "db_error")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxImageSize)
	if err := r.ParseMultipartForm(maxImageSize); err != nil {
		writeError(w, http.StatusBadRequest, "image too large (max 5MB)", "file_too_large")
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing image field", "missing_field")
		return
	}
	defer file.Close()

	// Validate content type.
	ext := strings.ToLower(filepath.Ext(header.Filename))
	validExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}
	if !validExts[ext] {
		writeError(w, http.StatusBadRequest, "image must be JPEG, PNG, or WebP", "invalid_file_type")
		return
	}

	// Build path.
	imagesDir := s.dataFilePath("images")
	if err := os.MkdirAll(imagesDir, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create images directory", "io_error")
		return
	}

	filename := fmt.Sprintf("device-%d-%d%s", id, time.Now().Unix(), ext)
	relPath := filepath.Join("images", filename)
	absPath := s.dataFilePath(relPath)

	dst, err := os.Create(absPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save image", "io_error")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		os.Remove(absPath)
		writeError(w, http.StatusInternalServerError, "failed to write image", "io_error")
		return
	}

	// Remove old image if present.
	if device.ImagePath != nil {
		os.Remove(s.dataFilePath(*device.ImagePath))
	}

	// Update device record.
	device.ImagePath = &relPath
	if err := s.sqlite.UpdateDevice(ctx, id, *device); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update device image", "db_error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"image_path": relPath})
}

// HandleDeviceImageDelete removes the image from a device.
func (s *Server) HandleDeviceImageDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := strconv.ParseInt(pathValue(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid device id", "invalid_param")
		return
	}

	device, err := s.sqlite.GetDevice(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "device not found", "not_found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch device", "db_error")
		return
	}

	if device.ImagePath != nil {
		os.Remove(s.dataFilePath(*device.ImagePath))
	}

	device.ImagePath = nil
	if err := s.sqlite.UpdateDevice(ctx, id, *device); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update device", "db_error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// dataFilePath returns an absolute path under the data directory (same directory as the SQLite file).
func (s *Server) dataFilePath(rel string) string {
	return filepath.Join(filepath.Dir(s.sqlite.Path()), rel)
}
