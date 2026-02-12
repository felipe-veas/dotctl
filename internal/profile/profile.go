package profile

import (
	"os"
	"runtime"
)

// Context holds the resolved runtime context used for manifest condition evaluation.
type Context struct {
	OS       string // runtime.GOOS: "darwin", "linux"
	Arch     string // runtime.GOARCH: "arm64", "amd64"
	Hostname string // os.Hostname()
	Profile  string // active profile name from config/flag
	Home     string // user home directory
}

// Resolve builds a Context from the current system state and the given profile name.
func Resolve(profile string) Context {
	hostname, _ := os.Hostname()
	home, _ := os.UserHomeDir()

	return Context{
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		Hostname: hostname,
		Profile:  profile,
		Home:     home,
	}
}

// Vars returns template variables available in manifest targets.
func (c Context) Vars() map[string]string {
	return map[string]string{
		"home":     c.Home,
		"os":       c.OS,
		"arch":     c.Arch,
		"profile":  c.Profile,
		"hostname": c.Hostname,
	}
}
