// Command github-exporter is a Prometheus exporter for GitHub self-hosted
// Actions runners, authenticated as a GitHub App installation.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"

	"github.com/flant/github-exporter/internal/agent"
	"github.com/flant/github-exporter/internal/config"
	"github.com/flant/github-exporter/internal/metrics"
	"github.com/flant/github-exporter/internal/router"
	"github.com/flant/github-exporter/internal/server"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if err := run(log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log.Info("configuration loaded",
		"org", cfg.Org,
		"app_id", cfg.AppID,
		"installation_id", cfg.InstallationID,
		"listen", cfg.ListenAddr,
		"metrics_path", cfg.MetricsPath,
	)

	gh, err := agent.New(cfg.AppID, cfg.InstallationID, cfg.PrivateKeyPath, cfg.Org, cfg.APIBaseURL)
	if err != nil {
		return err
	}

	scrapeCtx := func(parent context.Context) (context.Context, context.CancelFunc) {
		return context.WithTimeout(parent, cfg.ScrapeTimeout)
	}
	collector := metrics.New(cfg.Org, gh, scrapeCtx, log)

	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collector,
	)

	handler := router.New(reg, cfg.MetricsPath)
	srv := server.New(cfg.ListenAddr, handler, log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return srv.Run(ctx)
}
