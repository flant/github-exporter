package metrics

import (
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/flant/github-exporter/internal/agent"
)

// fakeSource stands in for the background poller's cached snapshot.
type fakeSource struct {
	runners []agent.Runner
	err     error
}

func (f *fakeSource) Runners() ([]agent.Runner, error) {
	return f.runners, f.err
}

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestCollectSuccess(t *testing.T) {
	c := New("acme", &fakeSource{runners: []agent.Runner{
		{ID: 1, Name: "r1", OS: "linux", Status: "online", Busy: true},
		{ID: 2, Name: "r2", OS: "linux", Status: "online", Busy: false},
		{ID: 3, Name: "r3", OS: "linux", Status: "offline", Busy: false},
	}}, quietLogger())

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
# HELP github_scrape_success 1 if the last poll of the GitHub API succeeded, 0 otherwise.
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
	c := New("acme", &fakeSource{runners: []agent.Runner{
		{ID: 1, Name: "r1", OS: "linux", Status: "online", Busy: true},
		{ID: 2, Name: "r2", OS: "windows", Status: "offline", Busy: false},
	}}, quietLogger())

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

func TestCollectStaleSnapshotSetsScrapeZero(t *testing.T) {
	c := New("acme", &fakeSource{err: errors.New("api down")}, quietLogger())

	expected := `
# HELP github_scrape_success 1 if the last poll of the GitHub API succeeded, 0 otherwise.
# TYPE github_scrape_success gauge
github_scrape_success{org="acme"} 0
`
	if err := testutil.CollectAndCompare(c, strings.NewReader(expected), "github_scrape_success"); err != nil {
		t.Error(err)
	}
	// With no usable snapshot, no per-runner or total metrics should be emitted.
	if n := testutil.CollectAndCount(c, "github_runners_total"); n != 0 {
		t.Errorf("expected no github_runners_total without a snapshot, got %d", n)
	}
}
