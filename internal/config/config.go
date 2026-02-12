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
	// DefaultRepoName is used when no explicit repo name is provided.
	DefaultRepoName = "default"
	// DefaultBackupKeep is the number of backup snapshots retained by default.
	DefaultBackupKeep = 20
)

// Config represents the local dotctl configuration stored on each machine.
type Config struct {
	// Legacy single-repo field kept for backward compatibility.
	Repo RepoConfig `yaml:"repo,omitempty"`
	// Multi-repo configuration.
	Repos      []RepoConfig `yaml:"repos,omitempty"`
	ActiveRepo string       `yaml:"active_repo,omitempty"`

	Profile  string       `yaml:"profile"`
	Backup   BackupConfig `yaml:"backup,omitempty"`
	LastSync *time.Time   `yaml:"last_sync,omitempty"`
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

	cfg.applyDefaults()

	return &cfg, nil
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

// Active returns the currently selected repository config.
func (c *Config) Active() (RepoConfig, error) {
	c.applyDefaults()

	if len(c.Repos) == 0 {
		return RepoConfig{}, errors.New("no repositories configured")
	}

	for _, repo := range c.Repos {
		if repo.Name == c.ActiveRepo {
			return repo, nil
		}
	}
	return RepoConfig{}, fmt.Errorf("active repo %q not found", c.ActiveRepo)
}

// SetActiveRepo switches the active repository by name.
func (c *Config) SetActiveRepo(name string) error {
	c.applyDefaults()

	normalized := NormalizeRepoName(name)
	if normalized == "" {
		return errors.New("repo name cannot be empty")
	}

	for _, repo := range c.Repos {
		if repo.Name == normalized {
			c.ActiveRepo = normalized
			c.Repo = repo
			return nil
		}
	}
	return fmt.Errorf("repo %q not found", normalized)
}

// UpsertRepo inserts a new repo or updates an existing one with the same name.
// It returns true when an existing repo was updated.
func (c *Config) UpsertRepo(repo RepoConfig) (bool, error) {
	c.applyDefaults()

	repo.Name = NormalizeRepoName(repo.Name)
	if repo.Name == "" {
		repo.Name = DefaultRepoName
	}
	repo.URL = strings.TrimSpace(repo.URL)
	if repo.URL == "" {
		return false, errors.New("repo URL cannot be empty")
	}
	if strings.TrimSpace(repo.Path) == "" {
		repo.Path = DefaultRepoPath(repo.Name)
	}

	for i := range c.Repos {
		if c.Repos[i].Name == repo.Name {
			c.Repos[i].URL = repo.URL
			c.Repos[i].Path = repo.Path
			if c.ActiveRepo == repo.Name {
				c.Repo = c.Repos[i]
			}
			return true, nil
		}
	}

	c.Repos = append(c.Repos, repo)
	if c.ActiveRepo == "" {
		c.ActiveRepo = repo.Name
	}
	if c.ActiveRepo == repo.Name {
		c.Repo = repo
	}
	return false, nil
}

// RemoveRepo deletes a repo by name.
func (c *Config) RemoveRepo(name string) error {
	c.applyDefaults()

	normalized := NormalizeRepoName(name)
	if normalized == "" {
		return errors.New("repo name cannot be empty")
	}

	next := make([]RepoConfig, 0, len(c.Repos))
	removed := false
	for _, repo := range c.Repos {
		if repo.Name == normalized {
			removed = true
			continue
		}
		next = append(next, repo)
	}

	if !removed {
		return fmt.Errorf("repo %q not found", normalized)
	}
	if len(next) == 0 {
		return errors.New("at least one repo must remain configured")
	}

	c.Repos = next
	if c.ActiveRepo == normalized {
		c.ActiveRepo = c.Repos[0].Name
	}
	active, _ := c.Active()
	c.Repo = active
	return nil
}

// NormalizeRepoName converts user input into a stable repo name.
func NormalizeRepoName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	name = strings.ReplaceAll(name, " ", "-")
	return name
}

// DefaultRepoPath returns the default clone directory for a repo name.
func DefaultRepoPath(name string) string {
	name = NormalizeRepoName(name)
	if name == "" || name == DefaultRepoName {
		return platform.RepoDir()
	}
	return filepath.Join(platform.ConfigDir(), "repo-"+name)
}

func (c *Config) applyDefaults() {
	if c.Backup.Keep <= 0 {
		c.Backup.Keep = DefaultBackupKeep
	}

	// Migrate legacy single-repo config to repos list.
	if len(c.Repos) == 0 && (strings.TrimSpace(c.Repo.URL) != "" || strings.TrimSpace(c.Repo.Path) != "") {
		legacy := c.Repo
		legacy.Name = NormalizeRepoName(legacy.Name)
		if legacy.Name == "" {
			legacy.Name = DefaultRepoName
		}
		if strings.TrimSpace(legacy.Path) == "" {
			legacy.Path = DefaultRepoPath(legacy.Name)
		}
		c.Repos = []RepoConfig{legacy}
	}

	if len(c.Repos) == 0 {
		// Keep sane defaults for brand-new config instances not yet initialized.
		if strings.TrimSpace(c.Repo.Path) == "" {
			c.Repo.Path = platform.RepoDir()
		}
		if strings.TrimSpace(c.Repo.Name) == "" {
			c.Repo.Name = DefaultRepoName
		}
		return
	}

	seen := make(map[string]bool, len(c.Repos))
	for i := range c.Repos {
		repo := &c.Repos[i]
		repo.Name = NormalizeRepoName(repo.Name)
		if repo.Name == "" {
			if i == 0 {
				repo.Name = DefaultRepoName
			} else {
				repo.Name = fmt.Sprintf("repo-%d", i+1)
			}
		}
		if seen[repo.Name] {
			base := repo.Name
			n := 2
			for seen[fmt.Sprintf("%s-%d", base, n)] {
				n++
			}
			repo.Name = fmt.Sprintf("%s-%d", base, n)
		}
		seen[repo.Name] = true

		repo.URL = strings.TrimSpace(repo.URL)
		if strings.TrimSpace(repo.Path) == "" {
			repo.Path = DefaultRepoPath(repo.Name)
		}
	}

	c.ActiveRepo = NormalizeRepoName(c.ActiveRepo)
	if c.ActiveRepo == "" || !seen[c.ActiveRepo] {
		c.ActiveRepo = c.Repos[0].Name
	}

	for _, repo := range c.Repos {
		if repo.Name == c.ActiveRepo {
			c.Repo = repo
			return
		}
	}
	c.Repo = c.Repos[0]
}
