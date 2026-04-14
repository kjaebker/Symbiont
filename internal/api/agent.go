package api

import (
	"net/http"
	"strings"

	"github.com/kjaebker/symbiont/internal/agent"
	"github.com/kjaebker/symbiont/internal/db"
)

var validTones = map[string]bool{
	"analytical": true,
	"casual":     true,
	"terse":      true,
}

var validDosingProductLines = map[string]bool{
	"brs_pharma":    true,
	"red_sea":       true,
	"tropic_marin":  true,
	"generic":       true,
	"none":          true,
}

// HandleAgentSettingsGet returns the current agent settings.
//
// GET /api/agent/settings
func (s *Server) HandleAgentSettingsGet(w http.ResponseWriter, r *http.Request) {
	settings, err := s.sqlite.GetAgentSettings(r.Context())
	if err != nil {
		s.logger.Error("failed to get agent settings", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get agent settings", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

// HandleAgentSettingsPut replaces the agent settings.
//
// PUT /api/agent/settings
func (s *Server) HandleAgentSettingsPut(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Tone              *string  `json:"tone"`
		DosingProductLine *string  `json:"dosing_product_line"`
		NetVolumeGallons  *float64 `json:"net_volume_gallons"`
		CustomGuardrails  *string  `json:"custom_guardrails"`
		EnabledSkills     []string `json:"enabled_skills"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "invalid_body")
		return
	}

	existing, err := s.sqlite.GetAgentSettings(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get agent settings", "db_error")
		return
	}

	// Patch fields that were provided.
	if body.Tone != nil {
		if !validTones[*body.Tone] {
			writeError(w, http.StatusBadRequest, "invalid tone; must be: analytical, casual, terse", "invalid_param")
			return
		}
		existing.Tone = *body.Tone
	}
	if body.DosingProductLine != nil {
		if !validDosingProductLines[*body.DosingProductLine] {
			writeError(w, http.StatusBadRequest, "invalid dosing_product_line; must be: brs_pharma, red_sea, tropic_marin, generic, none", "invalid_param")
			return
		}
		existing.DosingProductLine = *body.DosingProductLine
	}
	if body.NetVolumeGallons != nil {
		existing.NetVolumeGallons = body.NetVolumeGallons
	}
	if body.CustomGuardrails != nil {
		existing.CustomGuardrails = body.CustomGuardrails
	}
	if body.EnabledSkills != nil {
		existing.EnabledSkills = body.EnabledSkills
	}

	if err := s.sqlite.UpsertAgentSettings(r.Context(), *existing); err != nil {
		s.logger.Error("failed to upsert agent settings", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to save agent settings", "db_error")
		return
	}

	updated, err := s.sqlite.GetAgentSettings(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch agent settings", "db_error")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// HandleAgentContext assembles and returns the full agent context markdown.
//
// GET /api/agent/context
func (s *Server) HandleAgentContext(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	settings, err := s.sqlite.GetAgentSettings(ctx)
	if err != nil {
		s.logger.Error("failed to get agent settings", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get agent settings", "db_error")
		return
	}

	display, err := s.sqlite.GetTankProfile(ctx, "display")
	if err != nil {
		s.logger.Error("failed to get display profile", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get tank profile", "db_error")
		return
	}

	sump, err := s.sqlite.GetTankProfile(ctx, "sump")
	if err != nil {
		s.logger.Error("failed to get sump profile", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get tank profile", "db_error")
		return
	}

	livestock, err := s.sqlite.ListLivestock(ctx, db.LivestockFilter{Status: "active"})
	if err != nil {
		s.logger.Error("failed to list livestock", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get livestock", "db_error")
		return
	}

	alertRules, err := s.sqlite.ListAlertRules(ctx)
	if err != nil {
		s.logger.Error("failed to list alert rules", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get alert rules", "db_error")
		return
	}

	// Load enabled skills (all skills if none are explicitly listed).
	allSkills, err := agent.LoadSkills()
	if err != nil {
		s.logger.Error("failed to load skills", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to load skills", "skills_error")
		return
	}
	enabledSkills := filterEnabledSkills(allSkills, settings.EnabledSkills)

	markdown := agent.BuildContext(agent.ContextInput{
		Settings:   settings,
		Display:    display,
		Sump:       sump,
		Livestock:  livestock,
		AlertRules: alertRules,
		Skills:     enabledSkills,
	})

	// Accept: text/plain → return raw markdown; default → JSON envelope.
	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "text/plain") {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(markdown)) //nolint:errcheck
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"context": markdown})
}

// HandleAgentSkillList returns metadata for all available skills plus their
// enabled state from agent settings.
//
// GET /api/agent/skills
func (s *Server) HandleAgentSkillList(w http.ResponseWriter, r *http.Request) {
	allSkills, err := agent.LoadSkills()
	if err != nil {
		s.logger.Error("failed to load skills", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to load skills", "skills_error")
		return
	}

	settings, err := s.sqlite.GetAgentSettings(r.Context())
	if err != nil {
		s.logger.Error("failed to get agent settings", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get agent settings", "db_error")
		return
	}

	enabledSet := skillEnabledSet(settings.EnabledSkills)

	type skillResponse struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Enabled     bool   `json:"enabled"`
	}
	resp := make([]skillResponse, 0, len(allSkills))
	for _, sk := range allSkills {
		resp = append(resp, skillResponse{
			Name:        sk.Name,
			Description: sk.Description,
			Enabled:     enabledSet[sk.Name],
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"skills": resp})
}

// HandleAgentSkillBody returns the complete file content for a single skill.
//
// GET /api/agent/skills/{name}/body
func (s *Server) HandleAgentSkillBody(w http.ResponseWriter, r *http.Request) {
	name := pathValue(r, "name")
	sk, err := agent.GetSkill(name)
	if err != nil {
		writeError(w, http.StatusNotFound, "skill not found", "not_found")
		return
	}

	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "text/plain") {
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(sk.SkillFile())) //nolint:errcheck
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"name": sk.Name, "body": sk.SkillFile()})
}

// filterEnabledSkills returns the subset of skills that are enabled.
// If enabledNames is empty, all skills are returned (opt-out model).
func filterEnabledSkills(all []agent.Skill, enabledNames []string) []agent.Skill {
	if len(enabledNames) == 0 {
		return all
	}
	set := skillEnabledSet(enabledNames)
	out := make([]agent.Skill, 0, len(enabledNames))
	for _, sk := range all {
		if set[sk.Name] {
			out = append(out, sk)
		}
	}
	return out
}

func skillEnabledSet(names []string) map[string]bool {
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return m
}
