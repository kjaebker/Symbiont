package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mockAPI creates a test server that serves canned responses.
func mockAPI(t *testing.T, routes map[string]any) (*httptest.Server, *APIClient) {
	t.Helper()
	mux := http.NewServeMux()
	for pattern, response := range routes {
		resp := response // capture for closure
		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		})
	}
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	client := NewAPIClient(ts.URL, "test-token")
	return ts, client
}

func TestAPIClientGet(t *testing.T) {
	_, client := mockAPI(t, map[string]any{
		"GET /api/test": map[string]string{"hello": "world"},
	})

	var result map[string]string
	if err := client.Get(t.Context(), "/api/test", &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["hello"] != "world" {
		t.Fatalf("expected hello=world, got %v", result)
	}
}

func TestAPIClientError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "not found", "code": "not_found"})
	}))
	defer ts.Close()

	client := NewAPIClient(ts.URL, "test-token")
	err := client.Get(t.Context(), "/api/missing", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Status != 404 {
		t.Fatalf("expected status 404, got %d", apiErr.Status)
	}
}

func TestAPIClientPost(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": "1", "label": body["label"]})
	}))
	defer ts.Close()

	client := NewAPIClient(ts.URL, "test-token")
	var result map[string]string
	if err := client.Post(t.Context(), "/api/tokens", map[string]string{"label": "test"}, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["label"] != "test" {
		t.Fatalf("expected label=test, got %v", result)
	}
}

func TestAPIClientAuthHeader(t *testing.T) {
	var gotAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{})
	}))
	defer ts.Close()

	client := NewAPIClient(ts.URL, "my-secret-token")
	_ = client.Get(t.Context(), "/api/test", nil)
	if gotAuth != "Bearer my-secret-token" {
		t.Fatalf("expected 'Bearer my-secret-token', got %q", gotAuth)
	}
}

func TestPrintTable(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintTable([]string{"A", "B"}, [][]string{{"1", "2"}, {"3", "4"}})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "A") || !strings.Contains(output, "B") {
		t.Fatalf("expected headers in output, got %q", output)
	}
	if !strings.Contains(output, "1") || !strings.Contains(output, "4") {
		t.Fatalf("expected data in output, got %q", output)
	}
}

func TestPrintJSON(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintJSON(map[string]string{"key": "value"})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, `"key"`) || !strings.Contains(output, `"value"`) {
		t.Fatalf("expected JSON output, got %q", output)
	}
}

func TestTokenLoadFromEnv(t *testing.T) {
	t.Setenv("SYMBIONT_TOKEN", "env-token")
	token, err := LoadToken(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "env-token" {
		t.Fatalf("expected env-token, got %q", token)
	}
}

func TestTokenLoadFromFile(t *testing.T) {
	// Clear env to ensure file is used.
	t.Setenv("SYMBIONT_TOKEN", "")

	// Create a temp home dir with the token file.
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	dir := filepath.Join(tmpHome, ".config", "symbiont")
	os.MkdirAll(dir, 0o700)
	os.WriteFile(filepath.Join(dir, "token"), []byte("file-token\n"), 0o600)

	token, err := LoadToken(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "file-token" {
		t.Fatalf("expected file-token, got %q", token)
	}
}

func TestTokenLoadMissing(t *testing.T) {
	t.Setenv("SYMBIONT_TOKEN", "")
	t.Setenv("HOME", t.TempDir())

	_, err := LoadToken(nil)
	if err == nil {
		t.Fatal("expected error when no token found")
	}
}

func TestSaveToken(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	if err := SaveToken("saved-token"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpHome, ".config", "symbiont", "token"))
	if err != nil {
		t.Fatalf("reading token file: %v", err)
	}
	if strings.TrimSpace(string(data)) != "saved-token" {
		t.Fatalf("expected saved-token, got %q", string(data))
	}
}

func TestColorStatus(t *testing.T) {
	// Just verify it doesn't panic and returns something.
	if ColorStatus("normal") == "" {
		t.Fatal("expected non-empty string")
	}
	if ColorStatus("warning") == "" {
		t.Fatal("expected non-empty string")
	}
	if ColorStatus("critical") == "" {
		t.Fatal("expected non-empty string")
	}
	if ColorStatus("unknown") != "unknown" {
		t.Fatal("expected 'unknown' for unknown status")
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}
	for _, tt := range tests {
		result := formatBytes(tt.input)
		if result != tt.expected {
			t.Errorf("formatBytes(%d) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
