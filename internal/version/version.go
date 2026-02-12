package version

import "runtime"

// Version is set at build time via -ldflags.
var Version = "dev"

func OS() string {
	return runtime.GOOS
}

func Arch() string {
	return runtime.GOARCH
}

func Full() string {
	return Version + " (" + OS() + "/" + Arch() + ")"
}
