package linker

import (
	"os"
	"strings"
	"syscall"
	"testing"
)

func TestWrapPathErrorNoSpace(t *testing.T) {
	err := wrapPathError(
		"copying file",
		"/tmp/full-disk-target",
		&os.PathError{Op: "write", Path: "/tmp/full-disk-target", Err: syscall.ENOSPC},
	)
	got := strings.ToLower(err.Error())
	if !strings.Contains(got, "no space left on device") {
		t.Fatalf("expected no-space message, got: %v", err)
	}
	if !strings.Contains(got, "free disk space") {
		t.Fatalf("expected remediation hint, got: %v", err)
	}
}

func TestWrapPathErrorPermission(t *testing.T) {
	err := wrapPathError(
		"creating symlink",
		"/tmp/no-access",
		&os.PathError{Op: "symlink", Path: "/tmp/no-access", Err: syscall.EACCES},
	)
	got := strings.ToLower(err.Error())
	if !strings.Contains(got, "permission denied") {
		t.Fatalf("expected permission message, got: %v", err)
	}
}
