package api

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

// gzipPool reuses gzip writers; allocating one costs ~800KB of state.
var gzipPool = sync.Pool{
	New: func() any {
		w, _ := gzip.NewWriterLevel(io.Discard, gzip.DefaultCompression)
		return w
	},
}

// Compress gzips text-like responses (JSON, JS, CSS, HTML, SVG) when the
// client accepts it. The JS bundle and history JSON shrink 3–10x.
//
// Streaming endpoints (SSE, MCP) and Range requests pass through untouched,
// and the decision to compress is made per response from its Content-Type,
// so already-compressed bodies (images, downloads) are never re-encoded.
func Compress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !acceptsGzip(r) || r.Header.Get("Range") != "" ||
			r.URL.Path == "/api/stream" || r.URL.Path == "/api/mcp" {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Add("Vary", "Accept-Encoding")
		gw := &gzipResponseWriter{ResponseWriter: w}
		defer gw.close()
		next.ServeHTTP(gw, r)
	})
}

func acceptsGzip(r *http.Request) bool {
	for _, part := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
		enc, q, _ := strings.Cut(strings.TrimSpace(part), ";")
		if strings.EqualFold(strings.TrimSpace(enc), "gzip") {
			return strings.TrimSpace(q) != "q=0"
		}
	}
	return false
}

// compressibleType reports whether a Content-Type benefits from gzip.
func compressibleType(ct string) bool {
	ct, _, _ = strings.Cut(ct, ";")
	ct = strings.TrimSpace(strings.ToLower(ct))
	switch {
	case ct == "text/event-stream":
		return false
	case strings.HasPrefix(ct, "text/"):
		return true
	}
	switch ct {
	case "application/json", "application/javascript", "application/manifest+json",
		"image/svg+xml", "application/xml":
		return true
	}
	return false
}

type gzipResponseWriter struct {
	http.ResponseWriter
	gz          *gzip.Writer
	wroteHeader bool
}

func (g *gzipResponseWriter) WriteHeader(status int) {
	if g.wroteHeader {
		return
	}
	g.wroteHeader = true

	h := g.Header()
	if status >= 200 && status != http.StatusNoContent && status != http.StatusNotModified &&
		h.Get("Content-Encoding") == "" && compressibleType(h.Get("Content-Type")) {
		h.Del("Content-Length") // length of the uncompressed body; no longer valid
		h.Del("Accept-Ranges")
		h.Set("Content-Encoding", "gzip")
		gz := gzipPool.Get().(*gzip.Writer)
		gz.Reset(g.ResponseWriter)
		g.gz = gz
	}
	g.ResponseWriter.WriteHeader(status)
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	if !g.wroteHeader {
		if g.Header().Get("Content-Type") == "" {
			g.Header().Set("Content-Type", http.DetectContentType(b))
		}
		g.WriteHeader(http.StatusOK)
	}
	if g.gz != nil {
		return g.gz.Write(b)
	}
	return g.ResponseWriter.Write(b)
}

// Flush keeps http.Flusher working through the wrapper.
func (g *gzipResponseWriter) Flush() {
	if g.gz != nil {
		_ = g.gz.Flush()
	}
	if f, ok := g.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (g *gzipResponseWriter) close() {
	if g.gz == nil {
		return
	}
	_ = g.gz.Close() // writes the gzip trailer; client sees a truncated body on error regardless
	g.gz.Reset(io.Discard)
	gzipPool.Put(g.gz)
	g.gz = nil
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (g *gzipResponseWriter) Unwrap() http.ResponseWriter { return g.ResponseWriter }
