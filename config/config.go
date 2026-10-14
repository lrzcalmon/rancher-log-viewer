package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const DefaultMaxLines = 1000

type Config struct {
	RancherURL         string `json:"rancher_url"`
	Username           string `json:"username"`
	Password           string `json:"password"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify"`
	MaxLines           int    `json:"max_lines"`
}

func ConfigPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "config.json"
	}
	return filepath.Join(filepath.Dir(exe), "config.json")
}

func Exists() bool {
	path := ConfigPath()
	_, err := os.Stat(path)
	return err == nil
}

func Load() (*Config, error) {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config.json: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config.json: %w", err)
	}
	// Apply default if max_lines is missing or invalid
	if cfg.MaxLines < 200 || cfg.MaxLines > 5000 {
		cfg.MaxLines = DefaultMaxLines
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	path := ConfigPath()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config.json: %w", err)
	}
	return nil
}
