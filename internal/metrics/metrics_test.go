package metrics

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/flant/github-exporter/internal/agent"
)

type fakeClient struct {
	runners []agent.Runner
	err     error
}

func (f *fakeClient) ListRunners(context.Context) ([]agent.Runner, error) {
	return f.runners, f.err
}

func noTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(parent)
}

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestCollectSuccess(t *testing.T) {
	c := New("acme", &fakeClient{runners: []agent.Runner{
		{ID: 1, Name: "r1", OS: "linux", Status: "online", Busy: true},
		{ID: 2, Name: "r2", OS: "linux", Status: "online", Busy: false},
		{ID: 3, Name: "r3", OS: "linux", Status: "offline", Busy: false},
	}}, noTimeout, quietLogger())

	expected := `
# HELP github_runners_total Total number of self-hosted runners registered at the organization.
# TYPE github_runners_total gauge
github_runners_total{org="acme"} 3
# HELP github_runners_online_total Number of self-hosted runners currently online.
# TYPE github_runners_online_total gauge
github_runners_online_total{org="acme"} 2
# HELP github_runners_busy_total Number of self-hosted runners currently busy running a job.
# TYPE github_runners_busy_total gauge
github_runners_busy_total{org="acme"} 1
# HELP github_scrape_success 1 if the last scrape of the GitHub API succeeded, 0 otherwise.
# TYPE github_scrape_success gauge
github_scrape_success{org="acme"} 1
`
	if err := testutil.CollectAndCompare(c, strings.NewReader(expected),
		"github_runners_total", "github_runners_online_total",
		"github_runners_busy_total", "github_scrape_success"); err != nil {
		t.Error(err)
	}
}

func TestCollectPerRunnerMetrics(t *testing.T) {
	c := New("acme", &fakeClient{runners: []agent.Runner{
		{ID: 1, Name: "r1", OS: "linux", Status: "online", Busy: true},
		{ID: 2, Name: "r2", OS: "windows", Status: "offline", Busy: false},
	}}, noTimeout, quietLogger())

	expected := `
# HELP github_runner_status Runner connectivity: 1 if the runner is online, 0 otherwise.
# TYPE github_runner_status gauge
github_runner_status{name="r1",org="acme",os="linux",runner="1"} 1
github_runner_status{name="r2",org="acme",os="windows",runner="2"} 0
# HELP github_runner_busy Runner activity: 1 if the runner is currently running a job, 0 otherwise.
# TYPE github_runner_busy gauge
github_runner_busy{name="r1",org="acme",os="linux",runner="1"} 1
github_runner_busy{name="r2",org="acme",os="windows",runner="2"} 0
`
	if err := testutil.CollectAndCompare(c, strings.NewReader(expected),
		"github_runner_status", "github_runner_busy"); err != nil {
		t.Error(err)
	}
}

func TestCollectFailureSetsScrapeZero(t *testing.T) {
	c := New("acme", &fakeClient{err: errors.New("api down")}, noTimeout, quietLogger())

	expected := `
# HELP github_scrape_success 1 if the last scrape of the GitHub API succeeded, 0 otherwise.
# TYPE github_scrape_success gauge
github_scrape_success{org="acme"} 0
`
	if err := testutil.CollectAndCompare(c, strings.NewReader(expected), "github_scrape_success"); err != nil {
		t.Error(err)
	}
	// On failure no per-runner or total metrics should be emitted.
	if n := testutil.CollectAndCount(c, "github_runners_total"); n != 0 {
		t.Errorf("expected no github_runners_total on failure, got %d", n)
	}
}
