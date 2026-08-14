// Package router wires the exporter's HTTP endpoints together.
package router

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/flant/github-exporter/internal/handlers"
	"github.com/flant/github-exporter/internal/health"
)

// New returns an http.Handler serving the metrics endpoint (backed by the given
// registry), liveness (/healthz) and readiness (/readyz) probes, and a landing
// page. The readiness probe reflects the health state updated on each scrape.
func New(reg *prometheus.Registry, metricsPath string, state *health.State) http.Handler {
	mux := http.NewServeMux()
	mux.Handle(metricsPath, promhttp.HandlerFor(reg, promhttp.HandlerOpts{
		ErrorHandling: promhttp.ContinueOnError,
	}))
	mux.Handle("/healthz", handlers.Health())
	mux.Handle("/readyz", handlers.Ready(state))
	mux.Handle("/", handlers.Index(metricsPath))
	return mux
}
