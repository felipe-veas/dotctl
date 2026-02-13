package secrets

import (
	"path/filepath"
	"time"
)

// nowFunc is a variable for testing time-dependent code.
var nowFunc = time.Now

// dirOf returns the directory portion of a path, used for creating parent dirs.
func dirOf(path string) string {
	return filepath.Dir(path)
}
