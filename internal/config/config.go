package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/felipe-veas/dotctl/internal/platform"
	"gopkg.in/yaml.v3"
)

const (
	// DefaultBackupKeep is the number of backup snapshots retained by default.
	DefaultBackupKeep = 20
)

// Config represents the local dotctl configuration stored on each machine.
type Config struct {
	Repo     RepoConfig   `yaml:"repo,omitempty"`
	Backup   BackupConfig `yaml:"backup,omitempty"`
	LastSync *time.Time   `yaml:"last_sync,omitempty"`
}

type diskConfig struct {
	Repo       RepoConfig   `yaml:"repo,omitempty"`
	Repos      []RepoConfig `yaml:"repos,omitempty"`
	ActiveRepo string       `yaml:"active_repo,omitempty"`
	Profile    string       `yaml:"profile"` // deprecated; ignored during migration
	Backup     BackupConfig `yaml:"backup,omitempty"`
	LastSync   *time.Time   `yaml:"last_sync,omitempty"`
}

// RepoConfig holds the remote repository configuration.
type RepoConfig struct {
	Name string `yaml:"name,omitempty"`
	URL  string `yaml:"url"`
	Path string `yaml:"path"`
}

// BackupConfig controls backup retention behavior.
type BackupConfig struct {
	Keep int `yaml:"keep,omitempty"`
}

// DefaultPath returns the default config file path.
func DefaultPath() string {
	return filepath.Join(platform.ConfigDir(), "config.yaml")
}

// Load reads the config from the given path.
// Returns a zero Config and ErrNotFound if the file doesn't exist.
func Load(path string) (*Config, error) {
	safePath, err := normalizeConfigPath(path)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(safePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	var raw diskConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	if err := cfg.loadFromDisk(raw); err != nil {
		return nil, err
	}

	cfg.applyDefaults()

	return &cfg, nil
}

func (c *Config) loadFromDisk(raw diskConfig) error {
	c.Repo = raw.Repo
	c.Backup = raw.Backup
	c.LastSync = raw.LastSync

	if strings.TrimSpace(c.Repo.URL) != "" || strings.TrimSpace(c.Repo.Path) != "" {
		return nil
	}

	if len(raw.Repos) == 0 {
		return nil
	}
	if len(raw.Repos) == 1 {
		c.Repo = raw.Repos[0]
		return nil
	}

	active := NormalizeRepoName(raw.ActiveRepo)
	for _, repo := range raw.Repos {
		if NormalizeRepoName(repo.Name) == active && active != "" {
			c.Repo = repo
			return nil
		}
	}

	return fmt.Errorf("config has multiple deprecated repos; select one repository and keep it under the repo field before continuing")
}

func normalizeConfigPath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", errors.New("config path cannot be empty")
	}

	cleaned := filepath.Clean(trimmed)
	if cleaned == "." {
		return "", errors.New("config path cannot be empty")
	}
	if cleaned == ".." || strings.HasPrefix(filepath.ToSlash(cleaned), "../") {
		return "", fmt.Errorf("config path cannot use parent traversal: %q", path)
	}

	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return "", fmt.Errorf("resolving config path: %w", err)
	}

	return abs, nil
}

// Save writes the config to the given path, creating parent dirs as needed.
func Save(path string, cfg *Config) error {
	cfg.applyDefaults()

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

// SetRepo sets the single configured repository.
func (c *Config) SetRepo(repo RepoConfig) error {
	c.applyDefaults()

	repo.URL = strings.TrimSpace(repo.URL)
	if repo.URL == "" {
		return errors.New("repo URL cannot be empty")
	}
	if strings.TrimSpace(repo.Path) == "" {
		repo.Path = DefaultRepoPath()
	}
	c.Repo = repo
	return nil
}

// NormalizeRepoName converts user input into a stable repo name.
func NormalizeRepoName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	name = strings.ReplaceAll(name, " ", "-")
	return name
}

// DefaultRepoPath returns the default clone directory.
func DefaultRepoPath() string {
	return platform.RepoDir()
}

func (c *Config) applyDefaults() {
	if c.Backup.Keep <= 0 {
		c.Backup.Keep = DefaultBackupKeep
	}

	c.Repo.URL = strings.TrimSpace(c.Repo.URL)
	if strings.TrimSpace(c.Repo.Path) == "" {
		c.Repo.Path = DefaultRepoPath()
	}
}
