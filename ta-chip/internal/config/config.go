package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Credentials struct {
	LoginUser string `json:"login_user"`
	LoginPass string `json:"login_pass"`
}

type Config struct {
	GitHubRepo         string      `json:"github_repo"`
	AppScriptURL       string      `json:"appscript_url"`
	DomainName         string      `json:"domain_name"`
	DomainTestUser     string      `json:"domain_test_user"`
	DomainTestPassword string      `json:"domain_test_password"`
	ExpectedWallpaper  string      `json:"expected_wallpaper"`
	NTPToleranceSecs   int         `json:"ntp_tolerance_seconds"`
	Credentials        Credentials `json:"credentials"`
}

func Load() (*Config, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("cannot locate executable: %w", err)
	}
	path := filepath.Join(filepath.Dir(exe), "config.json")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config.json not found at %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config.json: %w", err)
	}

	// Sensible defaults
	if cfg.NTPToleranceSecs == 0 {
		cfg.NTPToleranceSecs = 300
	}
	if cfg.DomainName == "" {
		cfg.DomainName = "TECHLAB"
	}

	return &cfg, nil
}
