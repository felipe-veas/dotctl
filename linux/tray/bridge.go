//go:build linux && tray

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/felipe-veas/dotctl/pkg/types"
)

type dotctlBridge struct {
	binaryPath string
}

func newDotctlBridge() (*dotctlBridge, error) {
	if override := strings.TrimSpace(os.Getenv("DOTCTL_BIN")); override != "" {
		return &dotctlBridge{binaryPath: override}, nil
	}
	if lookup, err := exec.LookPath("dotctl"); err == nil {
		return &dotctlBridge{binaryPath: lookup}, nil
	}

	candidates := []string{
		"/usr/local/bin/dotctl",
		filepath.Join(os.Getenv("HOME"), ".local", "bin", "dotctl"),
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			return &dotctlBridge{binaryPath: candidate}, nil
		}
	}

	return nil, errors.New("dotctl binary not found (set DOTCTL_BIN or add dotctl to PATH)")
}

func (b *dotctlBridge) status(ctx context.Context) (types.StatusResponse, error) {
	data, err := b.run(ctx, "status", "--json")
	if err != nil {
		return types.StatusResponse{}, err
	}

	var status types.StatusResponse
	if err := json.Unmarshal(data, &status); err != nil {
		return types.StatusResponse{}, fmt.Errorf("parsing status JSON: %w", err)
	}

	return status, nil
}

func (b *dotctlBridge) sync(ctx context.Context) error {
	_, err := b.run(ctx, "sync", "--json")
	return err
}

func (b *dotctlBridge) pull(ctx context.Context) error {
	_, err := b.run(ctx, "pull", "--json")
	return err
}

func (b *dotctlBridge) push(ctx context.Context) error {
	_, err := b.run(ctx, "push", "--json")
	return err
}

func (b *dotctlBridge) doctor(ctx context.Context) error {
	_, err := b.run(ctx, "doctor", "--json")
	return err
}

func (b *dotctlBridge) openRepo(ctx context.Context) error {
	_, err := b.run(ctx, "open")
	return err
}

func (b *dotctlBridge) openConfig(ctx context.Context) error {
	configDir := dotctlConfigDir()
	cmd := exec.CommandContext(ctx, "xdg-open", configDir)
	combined, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("opening config dir %s: %s", configDir, strings.TrimSpace(string(combined)))
	}
	return nil
}

func (b *dotctlBridge) run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, b.binaryPath, args...)
	combined, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(combined))
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("%s %s failed: %s", b.binaryPath, strings.Join(args, " "), msg)
	}
	return combined, nil
}

func dotctlConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "dotctl")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".config", "dotctl")
	}
	return filepath.Join(home, ".config", "dotctl")
}
