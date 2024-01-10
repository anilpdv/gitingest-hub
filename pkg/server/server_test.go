package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestServerEndpoints(t *testing.T) {
	// Change working directory to project root for template resolution if needed
	os.Chdir("../../")
	defer os.Chdir("pkg/server")

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	t.Run("GET / returns 200 with HTML", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		srv.HandleIndex(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "GitIngest Hub") {
			t.Errorf("expected dashboard title in HTML response")
		}
	})

	t.Run("POST /api/ingest with empty URL returns error block", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/ingest", strings.NewReader("repo_url="))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		srv.HandleIngest(w, req)

		if !strings.Contains(w.Body.String(), "Ingestion Error") {
			t.Errorf("expected error block for empty repo URL")
		}
	})
}
