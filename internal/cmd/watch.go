package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/felipe-veas/dotctl/internal/output"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

func newWatchCmd() *cobra.Command {
	var debounce time.Duration
	var cooldown time.Duration

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch repo files and run sync automatically",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWatch(cmd, debounce, cooldown)
		},
	}

	cmd.Flags().DurationVar(&debounce, "debounce", 2*time.Second, "debounce window before triggering sync")
	cmd.Flags().DurationVar(&cooldown, "cooldown", 4*time.Second, "ignore filesystem events briefly after each sync")

	return cmd
}

func runWatch(cmd *cobra.Command, debounce, cooldown time.Duration) error {
	out := output.New(flagJSON)

	cfg, _, err := resolveConfig()
	if err != nil {
		return err
	}

	if debounce <= 0 {
		debounce = 2 * time.Second
	}
	if cooldown < 0 {
		cooldown = 0
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating fsnotify watcher: %w", err)
	}
	defer func() { _ = watcher.Close() }()

	if err := addWatchRecursive(watcher, cfg.Repo.Path); err != nil {
		return err
	}

	if !out.IsJSON() {
		out.Success("Watching %s (repo: %s)", cfg.Repo.Path, cfg.Repo.Name)
		out.Info("Debounce: %s | Cooldown: %s", debounce, cooldown)
		out.Info("Press Ctrl+C to stop.")
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	var pending bool
	var reason string
	var timer *time.Timer
	var timerC <-chan time.Time
	var running bool
	var suppressUntil time.Time

	triggerSync := func() {
		if running {
			pending = true
			return
		}
		running = true
		pending = false

		if !out.IsJSON() {
			out.Info("Change detected (%s), running sync...", reason)
		}
		syncErr := runSync(cmd, nil)
		if syncErr != nil {
			if !out.IsJSON() {
				out.Warn("Auto-sync failed: %v", syncErr)
			}
		} else if !out.IsJSON() {
			out.Success("Auto-sync complete.")
		}

		running = false
		suppressUntil = time.Now().Add(cooldown)
		if pending {
			pending = false
			if timer != nil {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(debounce)
				timerC = timer.C
			}
		}
	}

	schedule := func(triggerReason string) {
		reason = triggerReason
		pending = true
		if timer == nil {
			timer = time.NewTimer(debounce)
			timerC = timer.C
			return
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(debounce)
		timerC = timer.C
	}

	for {
		select {
		case <-signals:
			if !out.IsJSON() {
				out.Info("watch stopped")
			}
			return nil

		case event, ok := <-watcher.Events:
			if !ok {
				return errors.New("file watcher closed unexpectedly")
			}
			if ignoreWatchPath(event.Name, cfg.Repo.Path) {
				continue
			}
			if time.Now().Before(suppressUntil) {
				continue
			}

			if event.Op&fsnotify.Create == fsnotify.Create {
				info, statErr := os.Stat(event.Name)
				if statErr == nil && info.IsDir() {
					_ = addWatchRecursive(watcher, event.Name)
				}
			}

			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
				schedule(event.String())
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return errors.New("file watcher closed unexpectedly")
			}
			if !out.IsJSON() {
				out.Warn("watch error: %v", err)
			}

		case <-timerC:
			timerC = nil
			if pending {
				triggerSync()
			}
		}
	}
}

func addWatchRecursive(w *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, string(filepath.Separator)+".git") || filepath.Base(path) == ".git" {
			return filepath.SkipDir
		}
		if err := w.Add(path); err != nil {
			return fmt.Errorf("watch %s: %w", path, err)
		}
		return nil
	})
}

func ignoreWatchPath(path, repoRoot string) bool {
	rel, err := filepath.Rel(repoRoot, path)
	if err != nil {
		return true
	}
	rel = filepath.ToSlash(rel)
	return rel == ".git" || strings.HasPrefix(rel, ".git/")
}
