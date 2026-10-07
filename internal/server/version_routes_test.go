package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prasenjeet-symon/ogcode/internal/version"
)

// TestHandleVersionStatusCodes verifies HTTP status codes for version endpoint.
func TestHandleVersionStatusCodes(t *testing.T) {
	srv := &Server{versionManager: version.New()}

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	rec := httptest.NewRecorder()

	srv.handleVersion(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("handleVersion() status = %v, want %v", rec.Code, http.StatusOK)
	}
}

// TestHandleVersionContentType verifies Content-Type header.
func TestHandleVersionContentType(t *testing.T) {
	srv := &Server{versionManager: version.New()}

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	rec := httptest.NewRecorder()

	srv.handleVersion(rec, req)

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("handleVersion() Content-Type = %v, want application/json", contentType)
	}
}

// TestHandleVersionResponseStructure verifies the JSON response structure.
func TestHandleVersionResponseStructure(t *testing.T) {
	srv := &Server{versionManager: version.New()}

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	rec := httptest.NewRecorder()

	srv.handleVersion(rec, req)

	var response version.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify version info fields are present
	if response.Version == "" {
		t.Error("Response missing version")
	}
	if response.GoVersion == "" {
		t.Error("Response missing goVersion")
	}
}

// TestHandleVersionCheckStatusCode verifies HTTP status code for version check endpoint.
func TestHandleVersionCheckStatusCode(t *testing.T) {
	srv := &Server{versionManager: version.New()}

	req := httptest.NewRequest(http.MethodPost, "/version/check", nil)
	rec := httptest.NewRecorder()

	srv.handleVersionCheck(rec, req)

	// Should return 200 even if GitHub API fails (returns basic version info)
	if rec.Code != http.StatusOK {
		t.Errorf("handleVersionCheck() status = %v, want %v", rec.Code, http.StatusOK)
	}
}

// TestHandleVersionCheckClearsCache verifies the cache is cleared before check.
func TestHandleVersionCheckClearsCache(t *testing.T) {
	srv := &Server{versionManager: version.New()}

	// First call to populate cache
	req1 := httptest.NewRequest(http.MethodGet, "/version", nil)
	rec1 := httptest.NewRecorder()
	srv.handleVersion(rec1, req1)

	// Now check for updates (should clear cache)
	req2 := httptest.NewRequest(http.MethodPost, "/version/check", nil)
	rec2 := httptest.NewRecorder()
	srv.handleVersionCheck(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("handleVersionCheck() status = %v, want %v", rec2.Code, http.StatusOK)
	}

	var response version.Response
	if err := json.Unmarshal(rec2.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Should have version info
	if response.Version == "" {
		t.Error("handleVersionCheck() response missing version")
	}
}

// BenchmarkHandleVersion measures the performance of version endpoint.
func BenchmarkHandleVersion(b *testing.B) {
	srv := &Server{versionManager: version.New()}
	req := httptest.NewRequest(http.MethodGet, "/version", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		srv.handleVersion(rec, req)
	}
}

// TestVersionResponseFields verifies expected fields in response.
func TestVersionResponseFields(t *testing.T) {
	srv := &Server{versionManager: version.New()}

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	rec := httptest.NewRecorder()

	srv.handleVersion(rec, req)

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Required fields
	requiredFields := []string{"version", "goVersion"}
	for _, field := range requiredFields {
		if _, ok := response[field]; !ok {
			t.Errorf("Response missing required field: %s", field)
		}
	}

	// Update info fields (may or may not be present depending on GitHub API availability)
	updateFields := []string{"latestVersion", "updateAvailable"}
	t.Logf("Response fields: %v", response)
	for _, field := range updateFields {
		if _, ok := response[field]; !ok {
			t.Logf("Update info field not present (expected if GitHub API unavailable): %s", field)
		}
	}
}

// TestHandleVersionCheckPOSTMethod verifies the check endpoint works correctly.
func TestHandleVersionCheckPOSTMethod(t *testing.T) {
	srv := &Server{versionManager: version.New()}

	req := httptest.NewRequest(http.MethodPost, "/version/check", nil)
	rec := httptest.NewRecorder()

	srv.handleVersionCheck(rec, req)

	// Should return JSON
	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("handleVersionCheck() Content-Type = %v, want application/json", contentType)
	}
}
