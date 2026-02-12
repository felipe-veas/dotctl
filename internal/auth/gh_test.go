package auth

import (
	"errors"
	"strings"
	"testing"
)

func TestEnsureGHInstalledMissing(t *testing.T) {
	oldLookPath := lookPath
	lookPath = func(string) (string, error) {
		return "", errors.New("not found")
	}
	t.Cleanup(func() { lookPath = oldLookPath })

	err := EnsureGHInstalled()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "gh CLI not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnsureGHAuthenticatedOK(t *testing.T) {
	oldLookPath := lookPath
	oldRunCmd := runCmd
	lookPath = func(string) (string, error) {
		return "/usr/bin/gh", nil
	}
	runCmd = func(name string, args ...string) ([]byte, error) {
		if name != "gh" {
			t.Fatalf("name = %q, want gh", name)
		}
		return []byte("✓ Logged in to github.com account felipe-veas (keychain)\n"), nil
	}
	t.Cleanup(func() {
		lookPath = oldLookPath
		runCmd = oldRunCmd
	})

	user, err := EnsureGHAuthenticated()
	if err != nil {
		t.Fatalf("EnsureGHAuthenticated: %v", err)
	}
	if user != "felipe-veas" {
		t.Fatalf("user = %q, want felipe-veas", user)
	}
}

func TestEnsureGHAuthenticatedNotLoggedIn(t *testing.T) {
	oldLookPath := lookPath
	oldRunCmd := runCmd
	lookPath = func(string) (string, error) {
		return "/usr/bin/gh", nil
	}
	runCmd = func(string, ...string) ([]byte, error) {
		return []byte("not logged in"), errors.New("exit 1")
	}
	t.Cleanup(func() {
		lookPath = oldLookPath
		runCmd = oldRunCmd
	})

	_, err := EnsureGHAuthenticated()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "gh auth login --web") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "not logged in") {
		t.Fatalf("expected auth failure detail in error, got: %v", err)
	}
}

func TestExtractUser(t *testing.T) {
	user := extractUser("github.com\n  ✓ Logged in to github.com account octocat (keychain)\n")
	if user != "octocat" {
		t.Fatalf("user = %q, want octocat", user)
	}
}

func TestAuthFailureDetail(t *testing.T) {
	detail := authFailureDetail("\n\nnot logged in to any hosts\nmore")
	if detail != "not logged in to any hosts" {
		t.Fatalf("detail = %q", detail)
	}
}
