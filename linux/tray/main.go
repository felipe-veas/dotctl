//go:build linux && tray

package main

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/felipe-veas/dotctl/pkg/types"
	"github.com/getlantern/systray"
)

const (
	pollInterval       = 60 * time.Second
	statusTimeout      = 20 * time.Second
	defaultCmdTimeout  = 90 * time.Second
	syncCmdTimeout     = 10 * time.Minute
	maxErrorLineLength = 72
)

//go:embed assets/icon-ok.png
var iconOK []byte

//go:embed assets/icon-warn.png
var iconWarn []byte

//go:embed assets/icon-error.png
var iconError []byte

//go:embed assets/icon-sync.png
var iconSync []byte

type trayState string

const (
	stateOK      trayState = "ok"
	stateWarn    trayState = "warn"
	stateError   trayState = "error"
	stateSyncing trayState = "syncing"
)

type trayApp struct {
	bridge    *dotctlBridge
	bridgeErr error
	notifier  *notifier

	statusItem   *systray.MenuItem
	lastSyncItem *systray.MenuItem
	profileItem  *systray.MenuItem

	syncItem       *systray.MenuItem
	pullItem       *systray.MenuItem
	pushItem       *systray.MenuItem
	doctorItem     *systray.MenuItem
	openRepoItem   *systray.MenuItem
	openConfigItem *systray.MenuItem
	quitItem       *systray.MenuItem
}

func main() {
	bridge, err := newDotctlBridge()
	app := &trayApp{
		bridge:    bridge,
		bridgeErr: err,
		notifier:  newNotifier(),
	}
	systray.Run(app.onReady, app.onExit)
}

func (a *trayApp) onReady() {
	systray.SetTitle("dotctl")
	systray.SetTooltip("dotctl tray app")

	a.statusItem = systray.AddMenuItem("Status: loading...", "Current sync status")
	a.statusItem.Disable()
	a.lastSyncItem = systray.AddMenuItem("Last sync: --", "Last successful sync time")
	a.lastSyncItem.Disable()
	a.profileItem = systray.AddMenuItem("Profile: --", "Current dotctl profile")
	a.profileItem.Disable()

	systray.AddSeparator()
	a.syncItem = systray.AddMenuItem("Sync Now", "Run dotctl sync")
	a.pullItem = systray.AddMenuItem("Pull", "Run dotctl pull")
	a.pushItem = systray.AddMenuItem("Push", "Run dotctl push")
	a.doctorItem = systray.AddMenuItem("Doctor", "Run dotctl doctor")

	systray.AddSeparator()
	a.openRepoItem = systray.AddMenuItem("Open Repo", "Open repository in browser")
	a.openConfigItem = systray.AddMenuItem("Open Config", "Open dotctl config directory")

	systray.AddSeparator()
	a.quitItem = systray.AddMenuItem("Quit", "Close dotctl tray")

	if a.bridgeErr != nil {
		a.setState(stateError)
		a.setError(a.bridgeErr)
		a.disableActions(true)
		return
	}

	go a.runLoop()
}

func (a *trayApp) onExit() {}

func (a *trayApp) runLoop() {
	a.refreshStatus()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.refreshStatus()
		case <-a.syncItem.ClickedCh:
			a.runAction("sync", "syncing", syncCmdTimeout, a.bridge.sync, true, true)
		case <-a.pullItem.ClickedCh:
			a.runAction("pull", "pulling", defaultCmdTimeout, a.bridge.pull, true, true)
		case <-a.pushItem.ClickedCh:
			a.runAction("push", "pushing", defaultCmdTimeout, a.bridge.push, true, true)
		case <-a.doctorItem.ClickedCh:
			a.runAction("doctor", "running doctor", defaultCmdTimeout, a.bridge.doctor, true, true)
		case <-a.openRepoItem.ClickedCh:
			a.runAction("open repo", "opening repo", defaultCmdTimeout, a.bridge.openRepo, false, false)
		case <-a.openConfigItem.ClickedCh:
			a.runAction("open config", "opening config", defaultCmdTimeout, a.bridge.openConfig, false, false)
		case <-a.quitItem.ClickedCh:
			systray.Quit()
			return
		}
	}
}

func (a *trayApp) runAction(actionName, statusText string, timeout time.Duration, fn func(context.Context) error, refreshAfter, notify bool) {
	a.disableActions(true)
	a.statusItem.SetTitle("Status: " + statusText + "...")
	a.setState(stateSyncing)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := fn(ctx); err != nil {
		a.setError(err)
		if notify {
			a.notifier.notifyError(actionName, err)
		}
		a.disableActions(false)
		return
	}

	if notify {
		a.notifier.notifySuccess(actionName)
	}
	if refreshAfter {
		a.refreshStatus()
	}
	a.disableActions(false)
}

func (a *trayApp) refreshStatus() {
	ctx, cancel := context.WithTimeout(context.Background(), statusTimeout)
	defer cancel()

	status, err := a.bridge.status(ctx)
	if err != nil {
		a.setError(err)
		return
	}

	state, label := resolveState(status)
	a.setState(state)

	a.statusItem.SetTitle(fmt.Sprintf("Status: %s", label))
	lastSync := status.Repo.LastSync
	if lastSync == "" {
		lastSync = "never"
	}
	a.lastSyncItem.SetTitle("Last sync: " + lastSync)

	profile := status.Profile
	if profile == "" {
		profile = "(unset)"
	}
	a.profileItem.SetTitle("Profile: " + profile)
}

func (a *trayApp) setError(err error) {
	a.setState(stateError)
	a.statusItem.SetTitle("Status: Error")
	a.lastSyncItem.SetTitle("Error: " + shortLine(err.Error(), maxErrorLineLength))
}

func (a *trayApp) setState(state trayState) {
	switch state {
	case stateOK:
		systray.SetIcon(iconOK)
	case stateWarn:
		systray.SetIcon(iconWarn)
	case stateError:
		systray.SetIcon(iconError)
	default:
		systray.SetIcon(iconSync)
	}
}

func (a *trayApp) disableActions(disabled bool) {
	items := []*systray.MenuItem{
		a.syncItem,
		a.pullItem,
		a.pushItem,
		a.doctorItem,
		a.openRepoItem,
		a.openConfigItem,
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		if disabled {
			item.Disable()
		} else {
			item.Enable()
		}
	}
}

func resolveState(status types.StatusResponse) (trayState, string) {
	if len(status.Errors) > 0 || !status.Auth.OK || status.Symlinks.Broken > 0 || status.Repo.Status == "error" || status.Repo.Status == "not_git_repo" {
		return stateError, "error"
	}
	if status.Symlinks.Drift > 0 || status.Repo.Status == "dirty" {
		return stateWarn, fmt.Sprintf("drift (%d/%d ok)", status.Symlinks.OK, status.Symlinks.Total)
	}
	if status.Symlinks.Total == 0 {
		return stateWarn, "no managed symlinks"
	}
	return stateOK, fmt.Sprintf("synced (%d/%d ok)", status.Symlinks.OK, status.Symlinks.Total)
}

func shortLine(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
