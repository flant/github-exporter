// Package handlers provides HTTP handlers for the exporter's non-metrics
// endpoints (health check and landing page).
package handlers

import (
	"net/http"
)

// Health responds 200 OK for liveness/readiness probes.
func Health() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	}
}

// Index serves a minimal landing page linking to the metrics endpoint.
func Index(metricsPath string) http.HandlerFunc {
	body := []byte(`<html>
<head><title>GitHub Exporter</title></head>
<body>
<h1>GitHub Exporter</h1>
<p><a href="` + metricsPath + `">Metrics</a></p>
</body>
</html>`)
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(body)
	}
}
