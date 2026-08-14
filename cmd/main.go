// Command github-exporter is a Prometheus exporter for GitHub self-hosted
// Actions runners, authenticated as a GitHub App installation.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"

	"github.com/flant/github-exporter/internal/agent"
	"github.com/flant/github-exporter/internal/config"
	"github.com/flant/github-exporter/internal/health"
	"github.com/flant/github-exporter/internal/metrics"
	"github.com/flant/github-exporter/internal/poller"
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
		"poll_interval", cfg.PollInterval,
		"scrape_timeout", cfg.ScrapeTimeout,
	)

	gh, err := agent.New(cfg.AppID, cfg.InstallationID, cfg.PrivateKeyPath, cfg.Org, cfg.APIBaseURL)
	if err != nil {
		return err
	}

	// pollCtx bounds a single GitHub API request made by the poller.
	pollCtx := func(parent context.Context) (context.Context, context.CancelFunc) {
		return context.WithTimeout(parent, cfg.ScrapeTimeout)
	}

	healthState := health.New()
	p := poller.New(gh, healthState, cfg.PollInterval, pollCtx, log)
	collector := metrics.New(cfg.Org, p, log)

	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collector,
	)

	handler := router.New(reg, cfg.MetricsPath, healthState)
	srv := server.New(cfg.ListenAddr, handler, log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// The poller owns all GitHub API traffic; it stops when ctx is cancelled.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = p.Run(ctx)
	}()

	err = srv.Run(ctx)

	stop() // stop listening for signals, then unblock the poller
	wg.Wait()
	return err
}
