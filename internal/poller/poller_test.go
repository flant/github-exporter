package poller

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/flant/github-exporter/internal/agent"
	"github.com/flant/github-exporter/internal/health"
)

// fakeClient returns a scripted result per call, so a test can make the first
// poll succeed and the next one fail.
type fakeClient struct {
	mu      sync.Mutex
	calls   int
	runners [][]agent.Runner
	errs    []error
	called  chan struct{}
}

func (f *fakeClient) ListRunners(context.Context) ([]agent.Runner, error) {
	f.mu.Lock()
	i := f.calls
	f.calls++
	f.mu.Unlock()

	if f.called != nil {
		select {
		case f.called <- struct{}{}:
		default:
		}
	}

	var runners []agent.Runner
	if i < len(f.runners) {
		runners = f.runners[i]
	}
	var err error
	if i < len(f.errs) {
		err = f.errs[i]
	}
	return runners, err
}

func (f *fakeClient) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func noTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(parent)
}

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestRunnersBeforeFirstPoll(t *testing.T) {
	p := New(&fakeClient{}, health.New(), time.Second, noTimeout, quietLogger())

	runners, err := p.Runners()
	if err == nil {
		t.Error("expected an error before the first poll completes")
	}
	if runners != nil {
		t.Errorf("expected no runners before the first poll, got %v", runners)
	}
}

func TestPollSuccessCachesRunnersAndSetsReady(t *testing.T) {
	want := []agent.Runner{{ID: 1, Name: "r1", OS: "linux", Status: "online"}}
	h := health.New()
	p := New(&fakeClient{runners: [][]agent.Runner{want}}, h, time.Second, noTimeout, quietLogger())

	p.poll(context.Background())

	runners, err := p.Runners()
	if err != nil {
		t.Fatalf("unexpected error after a successful poll: %v", err)
	}
	if len(runners) != 1 || runners[0].Name != "r1" {
		t.Errorf("cached runners = %+v, want %+v", runners, want)
	}
	if ready, herr := h.Ready(); !ready {
		t.Errorf("health should be ready after a successful poll, err=%v", herr)
	}
}

func TestPollFailureSetsUnhealthyAndHidesStaleRunners(t *testing.T) {
	wantErr := errors.New("api down")
	h := health.New()
	p := New(&fakeClient{
		runners: [][]agent.Runner{{{ID: 1, Name: "r1"}}, nil},
		errs:    []error{nil, wantErr},
	}, h, time.Second, noTimeout, quietLogger())

	p.poll(context.Background()) // succeeds, caches r1
	p.poll(context.Background()) // fails

	runners, err := p.Runners()
	if !errors.Is(err, wantErr) {
		t.Errorf("snapshot error = %v, want %v", err, wantErr)
	}
	if runners != nil {
		t.Errorf("stale runners should not be exposed after a failed poll, got %v", runners)
	}
	if ready, _ := h.Ready(); ready {
		t.Error("health should be not-ready after a failed poll")
	}
}

func TestRunPollsImmediatelyAndStopsOnContextCancel(t *testing.T) {
	fc := &fakeClient{
		runners: [][]agent.Runner{{{ID: 1, Name: "r1"}}},
		called:  make(chan struct{}, 1),
	}
	// A long interval ensures the only poll we observe is the immediate one.
	p := New(fc, health.New(), time.Hour, noTimeout, quietLogger())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx) }()

	select {
	case <-fc.called:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not poll immediately on start")
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run returned %v, want nil on context cancel", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after context cancel")
	}

	if n := fc.callCount(); n != 1 {
		t.Errorf("expected exactly 1 poll with a long interval, got %d", n)
	}
}

func TestRunPollsOnInterval(t *testing.T) {
	fc := &fakeClient{called: make(chan struct{}, 1)}
	p := New(fc, health.New(), 10*time.Millisecond, noTimeout, quietLogger())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = p.Run(ctx) }()

	// Immediate poll plus at least one tick.
	for i := 0; i < 2; i++ {
		select {
		case <-fc.called:
		case <-time.After(2 * time.Second):
			t.Fatalf("expected poll %d to happen", i+1)
		}
	}
}
