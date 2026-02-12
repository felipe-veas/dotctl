package lock

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestAcquireExclusiveLock(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "sync.lock")

	first, err := Acquire(lockPath)
	if err != nil {
		t.Fatalf("first acquire failed: %v", err)
	}
	t.Cleanup(func() {
		_ = first.Release()
	})

	second, err := Acquire(lockPath)
	if err == nil {
		t.Fatalf("expected second acquire to fail, got lock: %+v", second)
	}
	if !errors.Is(err, ErrAlreadyLocked) {
		t.Fatalf("expected ErrAlreadyLocked, got: %v", err)
	}
}

func TestAcquireAfterRelease(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "sync.lock")

	l, err := Acquire(lockPath)
	if err != nil {
		t.Fatalf("acquire failed: %v", err)
	}
	if err := l.Release(); err != nil {
		t.Fatalf("release failed: %v", err)
	}

	l2, err := Acquire(lockPath)
	if err != nil {
		t.Fatalf("re-acquire failed: %v", err)
	}
	if err := l2.Release(); err != nil {
		t.Fatalf("second release failed: %v", err)
	}
}
