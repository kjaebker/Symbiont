package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestAgentSettingsGetDefaults(t *testing.T) {
	env := setupTestEnv(t)
	w := env.request(t, http.MethodGet, "/api/agent/settings", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["tone"] == nil {
		t.Error("expected tone in response")
	}
}

func TestAgentSettingsPutAndGet(t *testing.T) {
	env := setupTestEnv(t)

	w := env.request(t, http.MethodPut, "/api/agent/settings", map[string]any{
		"tone":                "casual",
		"dosing_product_line": "brs_pharma",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("PUT status = %d; body: %s", w.Code, w.Body.String())
	}

	w2 := env.request(t, http.MethodGet, "/api/agent/settings", nil)
	var resp map[string]any
	json.NewDecoder(w2.Body).Decode(&resp)
	if resp["tone"] != "casual" {
		t.Errorf("tone = %v, want casual", resp["tone"])
	}
	if resp["dosing_product_line"] != "brs_pharma" {
		t.Errorf("dosing_product_line = %v, want brs_pharma", resp["dosing_product_line"])
	}
}

func TestAgentSettingsPutInvalidTone(t *testing.T) {
	env := setupTestEnv(t)
	w := env.request(t, http.MethodPut, "/api/agent/settings", map[string]any{
		"tone": "super-sarcastic",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestAgentContextReturnsMarkdown(t *testing.T) {
	env := setupTestEnv(t)
	w := env.request(t, http.MethodGet, "/api/agent/context", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	ctx := resp["context"]
	if ctx == "" {
		t.Error("expected non-empty context")
	}
	if len(ctx) < 50 {
		t.Errorf("context suspiciously short: %q", ctx)
	}
}

func TestAgentSkillList(t *testing.T) {
	env := setupTestEnv(t)
	w := env.request(t, http.MethodGet, "/api/agent/skills", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Skills []map[string]any `json:"skills"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Skills) < 6 {
		t.Errorf("got %d skills, want at least 6", len(resp.Skills))
	}
	for _, s := range resp.Skills {
		if s["name"] == nil || s["description"] == nil {
			t.Errorf("skill missing name or description: %v", s)
		}
	}
}

func TestAgentSkillBody(t *testing.T) {
	env := setupTestEnv(t)
	w := env.request(t, http.MethodGet, "/api/agent/skills/water-test-analysis/body", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["body"] == "" {
		t.Error("expected non-empty skill body")
	}
}

func TestAgentSkillBodyNotFound(t *testing.T) {
	env := setupTestEnv(t)
	w := env.request(t, http.MethodGet, "/api/agent/skills/nonexistent/body", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}
