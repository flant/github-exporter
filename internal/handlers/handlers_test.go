package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flant/github-exporter/internal/health"
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

func TestReadyWhenHealthy(t *testing.T) {
	state := health.New()
	state.SetHealthy()

	rr := httptest.NewRecorder()
	Ready(state)(rr, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "ok") {
		t.Errorf("body = %q, want to contain \"ok\"", rr.Body.String())
	}
}

func TestReadyWhenUnhealthy(t *testing.T) {
	state := health.New() // never scraped successfully

	rr := httptest.NewRecorder()
	Ready(state)(rr, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "not ready") {
		t.Errorf("body = %q, want to contain \"not ready\"", rr.Body.String())
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
