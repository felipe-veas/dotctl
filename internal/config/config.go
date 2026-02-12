package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/felipe-veas/dotctl/internal/platform"
	"gopkg.in/yaml.v3"
)

// Config represents the local dotctl configuration stored on each machine.
type Config struct {
	Repo     RepoConfig `yaml:"repo"`
	Profile  string     `yaml:"profile"`
	LastSync *time.Time `yaml:"last_sync,omitempty"`
}

// RepoConfig holds the remote repository configuration.
type RepoConfig struct {
	URL  string `yaml:"url"`
	Path string `yaml:"path"`
}

// DefaultPath returns the default config file path.
func DefaultPath() string {
	return filepath.Join(platform.ConfigDir(), "config.yaml")
}

// Load reads the config from the given path.
// Returns a zero Config and ErrNotFound if the file doesn't exist.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Apply defaults
	if cfg.Repo.Path == "" {
		cfg.Repo.Path = platform.RepoDir()
	}

	return &cfg, nil
}

// Save writes the config to the given path, creating parent dirs as needed.
func Save(path string, cfg *Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

// Exists returns true if a config file exists at the given path.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ErrNotFound indicates no config file exists.
var ErrNotFound = errors.New("config file not found")
