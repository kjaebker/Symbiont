package api

import (
	"net/http"

	"github.com/kjaebker/symbiont/internal/db"
)

var validTankTypes = map[string]bool{
	"reef":       true,
	"fowlr":      true,
	"mixed":      true,
	"nano":       true,
	"freshwater": true,
	"other":      true,
}

// HandleTankProfileGet returns the profiles for both the display tank and sump.
// Either or both may be null if not yet configured.
//
// GET /api/tank/profile
func (s *Server) HandleTankProfileGet(w http.ResponseWriter, r *http.Request) {
	display, err := s.sqlite.GetTankProfile(r.Context(), "display")
	if err != nil {
		s.logger.Error("failed to get display tank profile", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get tank profile", "db_error")
		return
	}
	sump, err := s.sqlite.GetTankProfile(r.Context(), "sump")
	if err != nil {
		s.logger.Error("failed to get sump profile", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get tank profile", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"display": display,
		"sump":    sump,
	})
}

// HandleTankProfileUpsert returns a handler that upserts the tank profile for
// the given section ("display" or "sump").
//
// PUT /api/tank/profile/display
// PUT /api/tank/profile/sump
func (s *Server) HandleTankProfileUpsert(section string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			DisplayName   *string  `json:"display_name"`
			VolumeGallons *float64 `json:"volume_gallons"`
			LengthIn      *float64 `json:"length_in"`
			WidthIn       *float64 `json:"width_in"`
			HeightIn      *float64 `json:"height_in"`
			TankType      *string  `json:"tank_type"`
			Manufacturer  *string  `json:"manufacturer"`
			Model         *string  `json:"model"`
			SetupDate     *string  `json:"setup_date"`
			Notes         *string  `json:"notes"`
		}
		if err := readJSON(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body", "invalid_body")
			return
		}
		if body.TankType != nil && !validTankTypes[*body.TankType] {
			writeError(w, http.StatusBadRequest, "invalid tank_type; must be one of: reef, fowlr, mixed, nano, freshwater, other", "invalid_param")
			return
		}

		p := db.TankProfile{
			Section:       section,
			DisplayName:   body.DisplayName,
			VolumeGallons: body.VolumeGallons,
			LengthIn:      body.LengthIn,
			WidthIn:       body.WidthIn,
			HeightIn:      body.HeightIn,
			TankType:      body.TankType,
			Manufacturer:  body.Manufacturer,
			Model:         body.Model,
			SetupDate:     body.SetupDate,
			Notes:         body.Notes,
		}
		if err := s.sqlite.UpsertTankProfile(r.Context(), p); err != nil {
			s.logger.Error("failed to upsert tank profile", "section", section, "err", err)
			writeError(w, http.StatusInternalServerError, "failed to save tank profile", "db_error")
			return
		}

		updated, err := s.sqlite.GetTankProfile(r.Context(), section)
		if err != nil {
			s.logger.Error("failed to fetch updated tank profile", "section", section, "err", err)
			writeError(w, http.StatusInternalServerError, "failed to fetch tank profile", "db_error")
			return
		}
		writeJSON(w, http.StatusOK, updated)
	}
}
