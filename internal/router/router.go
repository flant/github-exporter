// Package router wires the exporter's HTTP endpoints together.
package router

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/flant/github-exporter/internal/handlers"
)

// New returns an http.Handler serving the metrics endpoint (backed by the given
// registry), a health check, and a landing page.
func New(reg *prometheus.Registry, metricsPath string) http.Handler {
	mux := http.NewServeMux()
	mux.Handle(metricsPath, promhttp.HandlerFor(reg, promhttp.HandlerOpts{
		ErrorHandling: promhttp.ContinueOnError,
	}))
	mux.Handle("/healthz", handlers.Health())
	mux.Handle("/", handlers.Index(metricsPath))
	return mux
}
