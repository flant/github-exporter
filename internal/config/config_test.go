package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// setEnv sets the given env vars for the duration of the test.
func setEnv(t *testing.T, vars map[string]string) {
	t.Helper()
	// Clear everything the loader reads so tests are isolated.
	for _, k := range []string{
		"APP_ID", "INSTALLATION_ID", "PRIVATE_KEY", "GITHUB_ORG",
		"LISTEN_ADDRESS", "METRICS_PATH", "GITHUB_API_URL", "SCRAPE_TIMEOUT",
		"POLL_INTERVAL",
	} {
		t.Setenv(k, "")
	}
	for k, v := range vars {
		t.Setenv(k, v)
	}
}

func writeKey(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "key.pem")
	if err := os.WriteFile(p, []byte("dummy"), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadDefaults(t *testing.T) {
	key := writeKey(t)
	setEnv(t, map[string]string{
		"APP_ID":          "123",
		"INSTALLATION_ID": "456",
		"PRIVATE_KEY":     key,
		"GITHUB_ORG":      "acme",
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AppID != 123 || cfg.InstallationID != 456 {
		t.Errorf("ids not parsed: %+v", cfg)
	}
	if cfg.ListenAddr != ":9101" {
		t.Errorf("default ListenAddr = %q", cfg.ListenAddr)
	}
	if cfg.MetricsPath != "/metrics" {
		t.Errorf("default MetricsPath = %q", cfg.MetricsPath)
	}
	if cfg.ScrapeTimeout != 30*time.Second {
		t.Errorf("default ScrapeTimeout = %v", cfg.ScrapeTimeout)
	}
	if cfg.PollInterval != 30*time.Second {
		t.Errorf("default PollInterval = %v", cfg.PollInterval)
	}
}

func TestLoadOverrides(t *testing.T) {
	key := writeKey(t)
	setEnv(t, map[string]string{
		"APP_ID":          "1",
		"INSTALLATION_ID": "2",
		"PRIVATE_KEY":     key,
		"GITHUB_ORG":      "acme",
		"LISTEN_ADDRESS":  ":8080",
		"METRICS_PATH":    "/m",
		"SCRAPE_TIMEOUT":  "5s",
		"POLL_INTERVAL":   "15s",
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ListenAddr != ":8080" || cfg.MetricsPath != "/m" || cfg.ScrapeTimeout != 5*time.Second {
		t.Errorf("overrides not applied: %+v", cfg)
	}
	if cfg.PollInterval != 15*time.Second {
		t.Errorf("PollInterval override not applied: %v", cfg.PollInterval)
	}
}

func TestLoadBadPollInterval(t *testing.T) {
	key := writeKey(t)
	setEnv(t, map[string]string{
		"APP_ID":          "1",
		"INSTALLATION_ID": "2",
		"PRIVATE_KEY":     key,
		"GITHUB_ORG":      "acme",
		"POLL_INTERVAL":   "notaduration",
	})
	if _, err := Load(); err == nil {
		t.Error("expected error for invalid POLL_INTERVAL")
	}
}

func TestLoadMissingRequired(t *testing.T) {
	cases := map[string]map[string]string{
		"no app id":          {"INSTALLATION_ID": "2", "GITHUB_ORG": "acme"},
		"no installation id": {"APP_ID": "1", "GITHUB_ORG": "acme"},
		"no org":             {"APP_ID": "1", "INSTALLATION_ID": "2"},
	}
	for name, vars := range cases {
		t.Run(name, func(t *testing.T) {
			key := writeKey(t)
			vars["PRIVATE_KEY"] = key
			setEnv(t, vars)
			if _, err := Load(); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestLoadBadAppID(t *testing.T) {
	key := writeKey(t)
	setEnv(t, map[string]string{
		"APP_ID":          "notanumber",
		"INSTALLATION_ID": "2",
		"PRIVATE_KEY":     key,
		"GITHUB_ORG":      "acme",
	})
	if _, err := Load(); err == nil {
		t.Error("expected error for non-integer APP_ID")
	}
}

func TestLoadMissingKeyFile(t *testing.T) {
	setEnv(t, map[string]string{
		"APP_ID":          "1",
		"INSTALLATION_ID": "2",
		"PRIVATE_KEY":     "/nonexistent/key.pem",
		"GITHUB_ORG":      "acme",
	})
	if _, err := Load(); err == nil {
		t.Error("expected error for missing key file")
	}
}
