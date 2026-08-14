// Package handlers provides HTTP handlers for the exporter's non-metrics
// endpoints (health check and landing page).
package handlers

import (
	"net/http"

	"github.com/flant/github-exporter/internal/health"
)

// Health responds 200 OK as long as the process is alive. It backs the
// liveness probe and deliberately does not depend on GitHub API reachability:
// a transient GitHub outage should not cause Kubernetes to restart the pod.
func Health() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	}
}

// Ready backs the readiness probe. It responds 200 while the last scrape of the
// GitHub API succeeded, and 503 (with the failure reason) when the exporter has
// not yet, or can no longer, list the organization's runners — taking the pod
// out of service rotation without restarting it.
func Ready(state *health.State) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if ready, err := state.Ready(); !ready {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("not ready: " + err.Error() + "\n"))
			return
		}
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
