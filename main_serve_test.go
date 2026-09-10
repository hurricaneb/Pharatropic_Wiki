package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func writeTestUpload(t *testing.T, name string, content []byte) {
	t.Helper()
	if err := os.MkdirAll("./uploads", 0755); err != nil {
		t.Fatalf("failed to ensure uploads dir: %v", err)
	}
	path := filepath.Join("./uploads", name)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("failed to write test upload %q: %v", name, err)
	}
	t.Cleanup(func() { os.Remove(path) })
}

func uploadRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/uploads/:filename", serveUploadFile)
	return r
}

func TestServeUploadFile_ImageIsInline(t *testing.T) {
	writeTestUpload(t, "test_image.png", []byte("\x89PNG\r\n\x1a\n rest of png bytes"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/uploads/test_image.png", nil)
	uploadRouter().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
	if got := w.Header().Get("Content-Disposition"); got != `inline; filename="test_image.png"` {
		t.Fatalf("Expected inline disposition for a PNG, got %q", got)
	}
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("Expected X-Content-Type-Options: nosniff on every response")
	}
}

func TestServeUploadFile_PDFIsInline(t *testing.T) {
	writeTestUpload(t, "test_doc.pdf", []byte("%PDF-1.4\n%fake pdf content\n"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/uploads/test_doc.pdf", nil)
	uploadRouter().ServeHTTP(w, req)

	if got := w.Header().Get("Content-Disposition"); got != `inline; filename="test_doc.pdf"` {
		t.Fatalf("Expected inline disposition for a PDF, got %q", got)
	}
}

func TestServeUploadFile_SVGForcesDownload(t *testing.T) {
	writeTestUpload(t, "test_evil.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/uploads/test_evil.svg", nil)
	uploadRouter().ServeHTTP(w, req)

	if got := w.Header().Get("Content-Disposition"); got != `attachment; filename="test_evil.svg"` {
		t.Fatalf("Expected attachment disposition for an SVG (XSS risk), got %q", got)
	}
}

func TestServeUploadFile_HTMLForcesDownload(t *testing.T) {
	writeTestUpload(t, "test_evil.html", []byte(`<html><body><script>alert(document.cookie)</script></body></html>`))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/uploads/test_evil.html", nil)
	uploadRouter().ServeHTTP(w, req)

	if got := w.Header().Get("Content-Disposition"); got != `attachment; filename="test_evil.html"` {
		t.Fatalf("Expected attachment disposition for HTML (XSS risk), got %q", got)
	}
}

func TestServeUploadFile_SpoofedExtensionStillSniffed(t *testing.T) {
	// A file named like an image but containing HTML must still be forced to
	// download: the decision is based on sniffed content, not the extension.
	writeTestUpload(t, "test_disguised.png", []byte(`<html><body><script>alert(1)</script></body></html>`))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/uploads/test_disguised.png", nil)
	uploadRouter().ServeHTTP(w, req)

	if got := w.Header().Get("Content-Disposition"); got != `attachment; filename="test_disguised.png"` {
		t.Fatalf("Expected attachment disposition for HTML disguised as .png, got %q", got)
	}
}

func TestServeUploadFile_NotFound(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/uploads/does-not-exist.png", nil)
	uploadRouter().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected 404 for a nonexistent file, got %d", w.Code)
	}
}

func TestServeUploadFile_PathTraversalIsNeutralized(t *testing.T) {
	// A traversal attempt must never serve a file outside ./uploads — either
	// the router itself refuses to match a multi-segment path against the
	// single-segment :filename param, or filepath.Base/the directory check
	// inside the handler catches it. Either way, this must not return 200.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/uploads/..%2F..%2F..%2Fetc%2Fpasswd", nil)
	uploadRouter().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected 404 for a path traversal attempt, got %d", w.Code)
	}
}

func TestServeUploadFile_DirectoryRejected(t *testing.T) {
	dirPath := filepath.Join("./uploads", "test_subdir")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dirPath) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/uploads/test_subdir", nil)
	uploadRouter().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected 404 requesting a directory, got %d", w.Code)
	}
}

// --- staticFileMiddleware ---

func staticRouter(distPath string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(staticFileMiddleware(distPath))
	r.NoRoute(func(c *gin.Context) { c.Status(http.StatusTeapot) }) // sentinel: middleware fell through
	r.GET("/api/v1/health", func(c *gin.Context) { c.String(http.StatusOK, "api-handled") })
	return r
}

func TestStaticFileMiddleware_DoesNotInterceptAPIRoutes(t *testing.T) {
	dist := t.TempDir()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/health", nil)
	staticRouter(dist).ServeHTTP(w, req)

	if w.Code != http.StatusOK || w.Body.String() != "api-handled" {
		t.Fatalf("Expected the API route to handle the request, got %d %q", w.Code, w.Body.String())
	}
}

func TestStaticFileMiddleware_ServesExistingFile(t *testing.T) {
	dist := t.TempDir()
	os.WriteFile(filepath.Join(dist, "app.js"), []byte("console.log('hi')"), 0644)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/app.js", nil)
	staticRouter(dist).ServeHTTP(w, req)

	if w.Code != http.StatusOK || w.Body.String() != "console.log('hi')" {
		t.Fatalf("Expected the static file to be served, got %d %q", w.Code, w.Body.String())
	}
}

func TestStaticFileMiddleware_FallsBackToIndexForSPARoutes(t *testing.T) {
	dist := t.TempDir()
	os.WriteFile(filepath.Join(dist, "index.html"), []byte("<html>spa shell</html>"), 0644)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/some/client/side/route", nil)
	staticRouter(dist).ServeHTTP(w, req)

	if w.Code != http.StatusOK || w.Body.String() != "<html>spa shell</html>" {
		t.Fatalf("Expected the SPA fallback to serve index.html, got %d %q", w.Code, w.Body.String())
	}
}

func TestStaticFileMiddleware_FallsThroughWhenNoIndexEither(t *testing.T) {
	dist := t.TempDir() // no index.html at all

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/unknown", nil)
	staticRouter(dist).ServeHTTP(w, req)

	if w.Code != http.StatusTeapot {
		t.Fatalf("Expected the request to fall through to NoRoute, got %d", w.Code)
	}
}
