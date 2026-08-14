// Package metrics implements a prometheus.Collector that exposes GitHub
// self-hosted runner state from the poller's cached snapshot.
package metrics

import (
	"log/slog"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/flant/github-exporter/internal/agent"
)

// runnerSource provides the most recent runner snapshot. It is satisfied by
// *poller.Poller; a non-nil error means the last poll failed, so the snapshot
// must not be exposed.
type runnerSource interface {
	Runners() ([]agent.Runner, error)
}

// Collector gathers runner metrics for a single organization. It implements
// prometheus.Collector and reads the snapshot maintained by the background
// poller, so a Prometheus scrape never hits the GitHub API itself.
type Collector struct {
	org    string
	source runnerSource
	log    *slog.Logger

	// Descriptors.
	runnerStatus *prometheus.Desc
	runnerBusy   *prometheus.Desc
	total        *prometheus.Desc
	onlineTotal  *prometheus.Desc
	busyTotal    *prometheus.Desc
	scrapeOK     *prometheus.Desc
}

// New builds a Collector reading runner state from src.
func New(org string, src runnerSource, log *slog.Logger) *Collector {
	labels := []string{"runner", "name", "os"}
	return &Collector{
		org:    org,
		source: src,
		log:    log,
		runnerStatus: prometheus.NewDesc(
			"github_runner_status",
			"Runner connectivity: 1 if the runner is online, 0 otherwise.",
			labels, prometheus.Labels{"org": org},
		),
		runnerBusy: prometheus.NewDesc(
			"github_runner_busy",
			"Runner activity: 1 if the runner is currently running a job, 0 otherwise.",
			labels, prometheus.Labels{"org": org},
		),
		total: prometheus.NewDesc(
			"github_runners_total",
			"Total number of self-hosted runners registered at the organization.",
			nil, prometheus.Labels{"org": org},
		),
		onlineTotal: prometheus.NewDesc(
			"github_runners_online_total",
			"Number of self-hosted runners currently online.",
			nil, prometheus.Labels{"org": org},
		),
		busyTotal: prometheus.NewDesc(
			"github_runners_busy_total",
			"Number of self-hosted runners currently busy running a job.",
			nil, prometheus.Labels{"org": org},
		),
		scrapeOK: prometheus.NewDesc(
			"github_scrape_success",
			"1 if the last poll of the GitHub API succeeded, 0 otherwise.",
			nil, prometheus.Labels{"org": org},
		),
	}
}

// Describe implements prometheus.Collector.
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.runnerStatus
	ch <- c.runnerBusy
	ch <- c.total
	ch <- c.onlineTotal
	ch <- c.busyTotal
	ch <- c.scrapeOK
}

// Collect implements prometheus.Collector. It reads the poller's latest
// snapshot; the readiness state is owned by the poller, not updated here.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	runners, err := c.source.Runners()
	if err != nil {
		c.log.Warn("serving metrics without runner data", "org", c.org, "err", err)
		ch <- prometheus.MustNewConstMetric(c.scrapeOK, prometheus.GaugeValue, 0)
		return
	}
	ch <- prometheus.MustNewConstMetric(c.scrapeOK, prometheus.GaugeValue, 1)

	var online, busy float64
	for _, r := range runners {
		id := strconv.FormatInt(r.ID, 10)
		ch <- prometheus.MustNewConstMetric(c.runnerStatus, prometheus.GaugeValue, boolToFloat(r.Online()), id, r.Name, r.OS)
		ch <- prometheus.MustNewConstMetric(c.runnerBusy, prometheus.GaugeValue, boolToFloat(r.Busy), id, r.Name, r.OS)
		if r.Online() {
			online++
		}
		if r.Busy {
			busy++
		}
	}

	ch <- prometheus.MustNewConstMetric(c.total, prometheus.GaugeValue, float64(len(runners)))
	ch <- prometheus.MustNewConstMetric(c.onlineTotal, prometheus.GaugeValue, online)
	ch <- prometheus.MustNewConstMetric(c.busyTotal, prometheus.GaugeValue, busy)
}

func boolToFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
