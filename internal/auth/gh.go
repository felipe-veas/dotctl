package auth

import (
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

var (
	lookPath = exec.LookPath
	runCmd   = runCommand

	ghUserPattern = regexp.MustCompile(`account\s+([^\s(]+)`) // "account <user>"
)

// Status represents GitHub authentication status for dotctl.
type Status struct {
	Method string
	User   string
	OK     bool
}

func runCommand(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.CombinedOutput()
}

// InstallHint returns the recommended gh installation command for the current OS.
func InstallHint() string {
	switch runtime.GOOS {
	case "darwin":
		return "brew install gh"
	case "linux":
		return "https://github.com/cli/cli/blob/trunk/docs/install_linux.md"
	default:
		return "https://github.com/cli/cli#installation"
	}
}

// EnsureGHInstalled verifies that the gh CLI is available.
func EnsureGHInstalled() error {
	if _, err := lookPath("gh"); err != nil {
		return fmt.Errorf("gh CLI not found — install: %s", InstallHint())
	}
	return nil
}

// EnsureGHAuthenticated verifies gh auth state and returns the authenticated user when available.
func EnsureGHAuthenticated() (string, error) {
	if err := EnsureGHInstalled(); err != nil {
		return "", err
	}

	out, err := runCmd("gh", "auth", "status")
	if err != nil {
		return "", fmt.Errorf("gh not authenticated — run: gh auth login --web")
	}

	return extractUser(string(out)), nil
}

// Check returns a structured authentication status for gh.
func Check() (Status, error) {
	user, err := EnsureGHAuthenticated()
	if err != nil {
		return Status{Method: "gh", OK: false}, err
	}

	return Status{
		Method: "gh",
		User:   user,
		OK:     true,
	}, nil
}

func extractUser(out string) string {
	matches := ghUserPattern.FindStringSubmatch(out)
	if len(matches) == 2 {
		return strings.TrimSpace(matches[1])
	}

	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Logged in to") {
			fields := strings.Fields(line)
			for i := 0; i < len(fields)-1; i++ {
				if fields[i] == "account" {
					return strings.Trim(fields[i+1], "()")
				}
			}
		}
	}

	return ""
}
