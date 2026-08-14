package health

import (
	"errors"
	"testing"
)

func TestNewStartsNotReady(t *testing.T) {
	s := New()
	ready, err := s.Ready()
	if ready {
		t.Error("fresh State should not be ready")
	}
	if err == nil {
		t.Error("fresh State should carry a reason for not being ready")
	}
}

func TestSetHealthy(t *testing.T) {
	s := New()
	s.SetHealthy()

	ready, err := s.Ready()
	if !ready {
		t.Error("State should be ready after SetHealthy")
	}
	if err != nil {
		t.Errorf("last error should be cleared after SetHealthy, got %v", err)
	}
}

func TestSetUnhealthy(t *testing.T) {
	s := New()
	s.SetHealthy()

	wantErr := errors.New("api down")
	s.SetUnhealthy(wantErr)

	ready, err := s.Ready()
	if ready {
		t.Error("State should not be ready after SetUnhealthy")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("last error = %v, want %v", err, wantErr)
	}
}
