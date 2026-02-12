package lock

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/felipe-veas/dotctl/internal/platform"
	"golang.org/x/sys/unix"
)

// ErrAlreadyLocked indicates another process already holds the lock.
var ErrAlreadyLocked = errors.New("another dotctl sync is already running")

// FileLock represents an acquired file lock.
type FileLock struct {
	path string
	file *os.File
}

// DefaultSyncLockPath returns the lock path used by dotctl sync.
func DefaultSyncLockPath() string {
	return filepath.Join(platform.StateDir(), "sync.lock")
}

// Acquire obtains an exclusive non-blocking flock on path.
func Acquire(path string) (*FileLock, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("creating lock directory: %w", err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("opening lock file: %w", err)
	}

	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, fmt.Errorf("%w (lock file: %s)", ErrAlreadyLocked, path)
		}
		return nil, fmt.Errorf("acquiring lock: %w", err)
	}

	_ = f.Truncate(0)
	_, _ = f.Seek(0, 0)
	_, _ = fmt.Fprintf(f, "%d\n", os.Getpid())

	return &FileLock{
		path: path,
		file: f,
	}, nil
}

// Path returns the lock file path.
func (l *FileLock) Path() string {
	if l == nil {
		return ""
	}
	return l.path
}

// Release unlocks and closes the lock file.
func (l *FileLock) Release() error {
	if l == nil || l.file == nil {
		return nil
	}

	unlockErr := unix.Flock(int(l.file.Fd()), unix.LOCK_UN)
	closeErr := l.file.Close()
	l.file = nil

	if unlockErr != nil {
		return fmt.Errorf("unlocking file lock: %w", unlockErr)
	}
	if closeErr != nil {
		return fmt.Errorf("closing lock file: %w", closeErr)
	}
	return nil
}
