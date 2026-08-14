// Package poller keeps a cached snapshot of an organization's self-hosted
// runners, refreshed on a fixed interval in the background.
//
// It is the exporter's single source of GitHub state: the metrics collector
// reads the cached snapshot instead of calling the API itself, and the
// readiness probe reflects the outcome of the latest poll. That keeps the load
// on the GitHub API constant regardless of how many Prometheus replicas scrape
// /metrics, and makes readiness independent of whether anything scrapes at all.
package poller

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/flant/github-exporter/internal/agent"
	"github.com/flant/github-exporter/internal/health"
)

// errNoPollYet is the snapshot error before the first poll completes.
var errNoPollYet = errors.New("no successful poll yet")

// runnerLister is the behavior the poller needs from the GitHub client.
type runnerLister interface {
	ListRunners(ctx context.Context) ([]agent.Runner, error)
}

// timeoutFunc bounds a single poll against the GitHub API. It is a field so
// tests can inject a context without a real deadline.
type timeoutFunc func(parent context.Context) (context.Context, context.CancelFunc)

// Poller periodically refreshes runner state and publishes it to readers.
type Poller struct {
	client   runnerLister
	health   *health.State
	interval time.Duration
	timeout  timeoutFunc
	log      *slog.Logger

	mu      sync.RWMutex
	runners []agent.Runner
	err     error
}

// New builds a Poller that refreshes every interval, bounding each GitHub
// request with pollCtx and reporting the outcome to h.
func New(client runnerLister, h *health.State, interval time.Duration, pollCtx timeoutFunc, log *slog.Logger) *Poller {
	return &Poller{
		client:   client,
		health:   h,
		interval: interval,
		timeout:  pollCtx,
		log:      log,
		err:      errNoPollYet,
	}
}

// Run polls immediately, then on every tick of the configured interval, until
// ctx is cancelled. It always returns nil: a failing poll is reported through
// the health state, not by terminating the exporter.
func (p *Poller) Run(ctx context.Context) error {
	p.log.Info("starting runner poller", "interval", p.interval)

	p.poll(ctx)

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.log.Info("stopping runner poller")
			return nil
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

// poll performs one refresh and updates both the cache and the health state.
func (p *Poller) poll(ctx context.Context) {
	pollCtx, cancel := p.timeout(ctx)
	defer cancel()

	runners, err := p.client.ListRunners(pollCtx)
	if err != nil {
		p.log.Error("poll failed", "err", err)
		p.mu.Lock()
		p.err = err
		p.mu.Unlock()
		p.health.SetUnhealthy(err)
		return
	}

	p.mu.Lock()
	p.runners = runners
	p.err = nil
	p.mu.Unlock()
	p.health.SetHealthy()
}

// Runners returns the most recent snapshot. The error is non-nil when the last
// poll failed (or none has completed yet), in which case the runners are stale
// and should not be exposed.
func (p *Poller) Runners() ([]agent.Runner, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.err != nil {
		return nil, p.err
	}
	return p.runners, nil
}
