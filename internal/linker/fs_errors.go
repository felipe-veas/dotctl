package linker

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"syscall"
)

func wrapPathError(operation, path string, err error) error {
	if err == nil {
		return nil
	}

	switch {
	case isSymlinkLoop(err):
		return fmt.Errorf("%s %q: symlink loop detected (fix circular links and retry): %w", operation, path, err)
	case isPermission(err):
		if runtime.GOOS == "darwin" {
			return fmt.Errorf("%s %q: permission denied (check file permissions and macOS privacy settings): %w", operation, path, err)
		}
		return fmt.Errorf("%s %q: permission denied (check ownership, permissions and writable parent directory): %w", operation, path, err)
	case isNoSpace(err):
		return fmt.Errorf("%s %q: no space left on device (free disk space and retry): %w", operation, path, err)
	default:
		return fmt.Errorf("%s %q: %w", operation, path, err)
	}
}

func isPermission(err error) bool {
	return errors.Is(err, os.ErrPermission) || errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM)
}

func isNoSpace(err error) bool {
	return errors.Is(err, syscall.ENOSPC)
}

func isSymlinkLoop(err error) bool {
	return errors.Is(err, syscall.ELOOP)
}
