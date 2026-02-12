//go:build linux && tray

package main

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

const notifyTimeout = 2 * time.Second

type notifier struct {
	enabled bool
}

func newNotifier() *notifier {
	_, err := exec.LookPath("notify-send")
	return &notifier{enabled: err == nil}
}

func (n *notifier) notifySuccess(action string) {
	if !n.enabled {
		return
	}
	title := "dotctl " + strings.TrimSpace(action)
	n.send(title, "completed successfully", "normal")
}

func (n *notifier) notifyError(action string, err error) {
	if !n.enabled {
		return
	}

	title := "dotctl " + strings.TrimSpace(action) + " failed"
	body := "unknown error"
	if err != nil {
		body = shortLine(err.Error(), 180)
	}
	n.send(title, body, "critical")
}

func (n *notifier) send(title, body, urgency string) {
	ctx, cancel := context.WithTimeout(context.Background(), notifyTimeout)
	defer cancel()

	args := []string{
		"--app-name=dotctl",
		"--urgency=" + strings.TrimSpace(urgency),
		strings.TrimSpace(title),
		strings.TrimSpace(body),
	}

	cmd := exec.CommandContext(ctx, "notify-send", args...)
	_, _ = cmd.CombinedOutput()
}
