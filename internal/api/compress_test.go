package api

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

// get issues a request with an explicit Accept-Encoding so Go's transport
// does not transparently decompress, letting us inspect the raw response.
func get(t *testing.T, url, acceptEncoding string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	if acceptEncoding != "" {
		req.Header.Set("Accept-Encoding", acceptEncoding)
	}
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })

	var body io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		zr, err := gzip.NewReader(resp.Body)
		if err != nil {
			t.Fatalf("gzip reader: %v", err)
		}
		body = zr
	}
	b, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	return resp, b
}

func TestCompress(t *testing.T) {
	bigJSON := `{"data":"` + strings.Repeat("abcdefgh", 2000) + `"}`

	mux := http.NewServeMux()
	mux.HandleFunc("/api/json", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, json.RawMessage(bigJSON))
	})
	mux.HandleFunc("/api/stream", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: hi\n\n")
	})
	mux.HandleFunc("/api/sse-elsewhere", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: hi\n\n")
	})
	mux.HandleFunc("/img", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("\x89PNG not really"))
	})
	mux.HandleFunc("/empty", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(Compress(mux))
	t.Cleanup(srv.Close)

	tests := []struct {
		name     string
		path     string
		accept   string
		wantGzip bool
		wantBody string
	}{
		{"json gzipped", "/api/json", "gzip, deflate, br", true, bigJSON + "\n"},
		{"json identity without accept", "/api/json", "", false, bigJSON + "\n"},
		{"json identity when gzip q=0", "/api/json", "gzip;q=0", false, bigJSON + "\n"},
		{"sse path bypassed", "/api/stream", "gzip", false, "data: hi\n\n"},
		{"event-stream type never compressed", "/api/sse-elsewhere", "gzip", false, "data: hi\n\n"},
		{"images not compressed", "/img", "gzip", false, "\x89PNG not really"},
		{"204 not compressed", "/empty", "gzip", false, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, body := get(t, srv.URL+tc.path, tc.accept)
			gotGzip := resp.Header.Get("Content-Encoding") == "gzip"
			if gotGzip != tc.wantGzip {
				t.Errorf("gzip = %v, want %v", gotGzip, tc.wantGzip)
			}
			if string(body) != tc.wantBody {
				t.Errorf("body mismatch: got %d bytes, want %d", len(body), len(tc.wantBody))
			}
		})
	}
}

func TestSPAHandlerCaching(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html":           {Data: []byte("<html>app</html>")},
		"assets/index-abc.js":  {Data: []byte(strings.Repeat("console.log(1);", 500))},
		"assets/index-abc.css": {Data: []byte("body{}")},
		"favicon.svg":          {Data: []byte("<svg/>")},
	}
	srv := httptest.NewServer(Compress(spaHandler(fsys)))
	t.Cleanup(srv.Close)

	tests := []struct {
		path      string
		wantCache string
		wantGzip  bool
	}{
		{"/assets/index-abc.js", "public, max-age=31536000, immutable", true},
		{"/assets/index-abc.css", "public, max-age=31536000, immutable", true},
		{"/", "no-cache", true},
		{"/livestock/42", "no-cache", true}, // SPA fallback to index.html
		{"/favicon.svg", "no-cache", true},
	}
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			resp, _ := get(t, srv.URL+tc.path, "gzip")
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status = %d", resp.StatusCode)
			}
			if got := resp.Header.Get("Cache-Control"); got != tc.wantCache {
				t.Errorf("Cache-Control = %q, want %q", got, tc.wantCache)
			}
			if got := resp.Header.Get("Content-Encoding") == "gzip"; got != tc.wantGzip {
				t.Errorf("gzip = %v, want %v", got, tc.wantGzip)
			}
		})
	}
}
