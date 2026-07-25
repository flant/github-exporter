// Package metrics implements a prometheus.Collector that scrapes GitHub
// self-hosted runner state on demand.
package metrics

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/flant/github-exporter/internal/agent"
)

// runnerLister is the behavior the collector needs from the GitHub client.
type runnerLister interface {
	ListRunners(ctx context.Context) ([]agent.Runner, error)
}

// Collector gathers runner metrics for a single organization. It implements
// prometheus.Collector and queries GitHub each time Prometheus scrapes it, so
// the exposed values always reflect the latest known state.
type Collector struct {
	org     string
	client  runnerLister
	timeout timeoutFunc
	log     *slog.Logger

	// Descriptors.
	runnerStatus *prometheus.Desc
	runnerBusy   *prometheus.Desc
	total        *prometheus.Desc
	onlineTotal  *prometheus.Desc
	busyTotal    *prometheus.Desc
	scrapeOK     *prometheus.Desc
}

// timeoutFunc derives a per-scrape context. It is a field so tests can inject a
// plain context.Background without a real deadline.
type timeoutFunc func(parent context.Context) (context.Context, context.CancelFunc)

// New builds a Collector. scrapeCtx wraps each scrape with a timeout.
func New(org string, client runnerLister, scrapeCtx timeoutFunc, log *slog.Logger) *Collector {
	labels := []string{"runner", "name", "os"}
	return &Collector{
		org:     org,
		client:  client,
		timeout: scrapeCtx,
		log:     log,
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
			"1 if the last scrape of the GitHub API succeeded, 0 otherwise.",
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

// Collect implements prometheus.Collector. It performs a live scrape.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := c.timeout(context.Background())
	defer cancel()

	runners, err := c.client.ListRunners(ctx)
	if err != nil {
		c.log.Error("scrape failed", "org", c.org, "err", err)
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
