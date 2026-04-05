package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSystemLayoutGetEmpty(t *testing.T) {
	env := setupTestEnv(t)

	w := env.request(t, "GET", "/api/system/layout", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if _, ok := resp["layout"]; !ok {
		t.Fatal("expected layout key in response")
	}
	if resp["layout"] != nil {
		t.Errorf("expected null layout, got %v", resp["layout"])
	}
}

func TestSystemLayoutPutAndGet(t *testing.T) {
	env := setupTestEnv(t)

	layout := map[string]any{
		"version":  1,
		"nodes":    []any{},
		"edges":    []any{},
		"viewport": map[string]any{"x": 0, "y": 0, "zoom": 1},
	}

	w := env.request(t, "PUT", "/api/system/layout", map[string]any{"layout": layout})
	if w.Code != http.StatusOK {
		t.Fatalf("PUT expected 200, got %d: %s", w.Code, w.Body.String())
	}

	w = env.request(t, "GET", "/api/system/layout", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	got, ok := resp["layout"].(map[string]any)
	if !ok {
		t.Fatalf("expected layout object, got %T: %v", resp["layout"], resp["layout"])
	}
	if got["version"] != float64(1) {
		t.Errorf("expected version 1, got %v", got["version"])
	}
}

func TestSystemLayoutOverwrite(t *testing.T) {
	env := setupTestEnv(t)

	env.request(t, "PUT", "/api/system/layout", map[string]any{
		"layout": map[string]any{"version": 1, "nodes": []any{}, "edges": []any{}},
	})

	newLayout := map[string]any{"version": 1, "nodes": []any{map[string]any{"id": "n1"}}, "edges": []any{}}
	env.request(t, "PUT", "/api/system/layout", map[string]any{"layout": newLayout})

	w := env.request(t, "GET", "/api/system/layout", nil)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	got := resp["layout"].(map[string]any)
	nodes := got["nodes"].([]any)
	if len(nodes) != 1 {
		t.Errorf("expected 1 node after overwrite, got %d", len(nodes))
	}
}

func TestSystemLayoutPutInvalidBody(t *testing.T) {
	env := setupTestEnv(t)

	// Send malformed JSON body directly.
	req := httptest.NewRequest("PUT", "/api/system/layout", bytes.NewBufferString(`not-json`))
	req.Header.Set("Authorization", "Bearer "+env.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.server.http.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for malformed body, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSystemLayoutPutMissingLayout(t *testing.T) {
	env := setupTestEnv(t)

	w := env.request(t, "PUT", "/api/system/layout", map[string]any{})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing layout, got %d: %s", w.Code, w.Body.String())
	}
}
