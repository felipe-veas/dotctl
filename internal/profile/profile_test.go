package profile

import (
	"runtime"
	"testing"
)

func TestResolve(t *testing.T) {
	ctx := Resolve("macstudio")

	if ctx.OS != runtime.GOOS {
		t.Errorf("OS = %q, want %q", ctx.OS, runtime.GOOS)
	}
	if ctx.Arch != runtime.GOARCH {
		t.Errorf("Arch = %q, want %q", ctx.Arch, runtime.GOARCH)
	}
	if ctx.Profile != "macstudio" {
		t.Errorf("Profile = %q, want %q", ctx.Profile, "macstudio")
	}
	if ctx.Home == "" {
		t.Error("Home should not be empty")
	}
	if ctx.Hostname == "" {
		t.Error("Hostname should not be empty")
	}
}

func TestVars(t *testing.T) {
	ctx := Context{
		OS:       "darwin",
		Arch:     "arm64",
		Hostname: "mac.local",
		Profile:  "macstudio",
		Home:     "/Users/test",
	}

	vars := ctx.Vars()

	expected := map[string]string{
		"home":     "/Users/test",
		"os":       "darwin",
		"arch":     "arm64",
		"profile":  "macstudio",
		"hostname": "mac.local",
	}

	for k, want := range expected {
		if got := vars[k]; got != want {
			t.Errorf("Vars[%q] = %q, want %q", k, got, want)
		}
	}
}
