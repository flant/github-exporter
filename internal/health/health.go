// Package health tracks whether the exporter is able to reach the GitHub API,
// exposing that state to Kubernetes readiness probes.
//
// The state is updated by the metrics collector on every scrape and read by the
// /readyz handler. A fresh State starts not-ready: the exporter only becomes
// ready once it has successfully listed the organization's runners at least
// once.
package health

import (
	"errors"
	"sync"
)

// State is a concurrency-safe readiness flag with the last observed error.
type State struct {
	mu      sync.RWMutex
	ready   bool
	lastErr error
}

// New returns a State in the not-ready state.
func New() *State {
	return &State{
		lastErr: errors.New("no successful scrape yet"),
	}
}

// SetHealthy marks the exporter ready and clears the last error.
func (s *State) SetHealthy() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ready = true
	s.lastErr = nil
}

// SetUnhealthy marks the exporter not ready and records why.
func (s *State) SetUnhealthy(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ready = false
	s.lastErr = err
}

// Ready reports whether the last scrape succeeded. When not ready it also
// returns the error that caused it, for surfacing in the probe response.
func (s *State) Ready() (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ready, s.lastErr
}
