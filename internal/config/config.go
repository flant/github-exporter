// Package config loads the exporter configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all runtime settings for the exporter.
type Config struct {
	// GitHub App authentication.
	AppID          int64
	InstallationID int64
	PrivateKeyPath string

	// Org whose self-hosted runners are scraped.
	Org string

	// HTTP server.
	ListenAddr  string
	MetricsPath string

	// APIBaseURL overrides the GitHub API base URL (for GitHub Enterprise
	// Server). Empty means public github.com.
	APIBaseURL string

	// ScrapeTimeout bounds a single collection against the GitHub API.
	ScrapeTimeout time.Duration
}

// Load reads configuration from the environment, applying defaults and
// validating required fields.
func Load() (*Config, error) {
	cfg := &Config{
		ListenAddr:     getEnv("LISTEN_ADDRESS", ":9101"),
		MetricsPath:    getEnv("METRICS_PATH", "/metrics"),
		Org:            os.Getenv("GITHUB_ORG"),
		PrivateKeyPath: os.Getenv("PRIVATE_KEY"),
		APIBaseURL:     os.Getenv("GITHUB_API_URL"),
		ScrapeTimeout:  30 * time.Second,
	}

	var err error
	if cfg.AppID, err = getEnvInt64("APP_ID"); err != nil {
		return nil, err
	}
	if cfg.InstallationID, err = getEnvInt64("INSTALLATION_ID"); err != nil {
		return nil, err
	}

	if v := os.Getenv("SCRAPE_TIMEOUT"); v != "" {
		d, perr := time.ParseDuration(v)
		if perr != nil {
			return nil, fmt.Errorf("invalid SCRAPE_TIMEOUT %q: %w", v, perr)
		}
		cfg.ScrapeTimeout = d
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	if c.AppID == 0 {
		return fmt.Errorf("APP_ID is required")
	}
	if c.InstallationID == 0 {
		return fmt.Errorf("INSTALLATION_ID is required")
	}
	if c.PrivateKeyPath == "" {
		return fmt.Errorf("PRIVATE_KEY (path to the GitHub App private key) is required")
	}
	if _, err := os.Stat(c.PrivateKeyPath); err != nil {
		return fmt.Errorf("PRIVATE_KEY file %q is not accessible: %w", c.PrivateKeyPath, err)
	}
	if c.Org == "" {
		return fmt.Errorf("GITHUB_ORG is required")
	}
	return nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt64(key string) (int64, error) {
	v := os.Getenv(key)
	if v == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: must be an integer", key, v)
	}
	return n, nil
}
