package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealth(t *testing.T) {
	rr := httptest.NewRecorder()
	Health()(rr, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "ok") {
		t.Errorf("body = %q, want to contain \"ok\"", rr.Body.String())
	}
}

func TestIndexRoot(t *testing.T) {
	rr := httptest.NewRecorder()
	Index("/metrics")(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `href="/metrics"`) {
		t.Errorf("index page should link to metrics path, got %q", rr.Body.String())
	}
}

func TestIndexNotFound(t *testing.T) {
	rr := httptest.NewRecorder()
	Index("/metrics")(rr, httptest.NewRequest(http.MethodGet, "/other", nil))

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 for unknown path", rr.Code)
	}
}
