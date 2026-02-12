package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// ConfigDir returns the dotctl config directory, respecting XDG_CONFIG_HOME.
func ConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "dotctl")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "dotctl")
}

// StateDir returns the dotctl state directory (logs), respecting XDG_STATE_HOME.
func StateDir() string {
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		return filepath.Join(xdg, "dotctl")
	}
	if runtime.GOOS == "linux" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".local", "state", "dotctl")
	}
	return ConfigDir()
}

// RepoDir returns the default path for cloning the dotfiles repo.
func RepoDir() string {
	return filepath.Join(ConfigDir(), "repo")
}

// BackupDir returns the base directory for backups.
func BackupDir() string {
	return filepath.Join(ConfigDir(), "backups")
}

// OpenURL opens a URL in the default browser.
func OpenURL(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Run()
	case "linux":
		return exec.Command("xdg-open", url).Run()
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// OpenFileManager opens a path in the system file manager.
func OpenFileManager(path string) error {
	return OpenURL(path)
}
