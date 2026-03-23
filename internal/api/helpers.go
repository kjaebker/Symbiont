package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const maxBodySize = 1 << 20 // 1MB

// writeJSON marshals v as JSON and writes it to w with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Best effort — headers already sent.
		_ = err
	}
}

// writeError writes a consistent JSON error response.
func writeError(w http.ResponseWriter, status int, msg, code string) {
	writeJSON(w, status, map[string]string{
		"error": msg,
		"code":  code,
	})
}

// readJSON decodes the request body into v with a size limit.
func readJSON(r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, maxBodySize)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("decoding request body: %w", err)
	}
	// Ensure only one JSON value in body.
	if dec.More() {
		return fmt.Errorf("request body contains multiple JSON values")
	}
	return nil
}

// queryParam returns the query parameter value or a default.
func queryParam(r *http.Request, key, defaultVal string) string {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	return v
}

// pathValue extracts a path value from the request using Go 1.22+ routing.
func pathValue(r *http.Request, key string) string {
	return r.PathValue(key)
}

// respondNotFound sends a standard 404 error.
func respondNotFound(w http.ResponseWriter) {
	writeError(w, http.StatusNotFound, "not found", "not_found")
}

// respondMethodNotAllowed sends a standard 405 error.
func respondMethodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, "method not allowed", "method_not_allowed")
}

// discardBody reads and discards the request body to allow connection reuse.
func discardBody(r *http.Request) {
	_, _ = io.Copy(io.Discard, r.Body)
	r.Body.Close()
}

// splitCamelCase inserts spaces before uppercase letters in a CamelCase string.
// "SumpFlow" → "Sump Flow", "ReturnPump" → "Return Pump", "pH" → "pH".
func splitCamelCase(s string) string {
	if len(s) <= 1 {
		return s
	}
	var result []byte
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if i > 0 && ch >= 'A' && ch <= 'Z' {
			// Don't insert space if previous char is also uppercase (acronym).
			prevUpper := s[i-1] >= 'A' && s[i-1] <= 'Z'
			if !prevUpper {
				result = append(result, ' ')
			}
		}
		result = append(result, ch)
	}
	return string(result)
}
