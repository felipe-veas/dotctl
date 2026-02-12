package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/felipe-veas/dotctl/internal/platform"
)

var (
	mu          sync.Mutex
	currentFile *os.File
	currentPath string
)

var tokenPattern = regexp.MustCompile(`\b(?:gh[psu]_[A-Za-z0-9_]{8,}|github_pat_[A-Za-z0-9_]{8,})\b`)

// Path returns the absolute path to the dotctl log file.
func Path() string {
	return filepath.Join(platform.StateDir(), "dotctl.log")
}

// Init configures slog to write JSON logs into the per-OS state directory.
func Init(verbose bool) error {
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating log directory: %w", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// Reuse the open file descriptor when the path is unchanged.
	if currentFile != nil && currentPath == path {
		setDefaultLogger(currentFile, verbose)
		return nil
	}

	if currentFile != nil {
		_ = currentFile.Close()
		currentFile = nil
		currentPath = ""
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}

	currentFile = f
	currentPath = path
	setDefaultLogger(currentFile, verbose)
	return nil
}

// Close closes the active log file if initialized.
func Close() error {
	mu.Lock()
	defer mu.Unlock()

	if currentFile == nil {
		return nil
	}

	err := currentFile.Close()
	currentFile = nil
	currentPath = ""
	return err
}

func setDefaultLogger(w io.Writer, verbose bool) {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			if attr.Value.Kind() == slog.KindString {
				attr.Value = slog.StringValue(redact(attr.Value.String()))
			}
			return attr
		},
	})

	slog.SetDefault(slog.New(handler))
}

// Debug writes a debug log entry.
func Debug(msg string, attrs ...any) {
	slog.Debug(redact(msg), sanitizeArgs(attrs)...)
}

// Info writes an info log entry.
func Info(msg string, attrs ...any) {
	slog.Info(redact(msg), sanitizeArgs(attrs)...)
}

// Warn writes a warning log entry.
func Warn(msg string, attrs ...any) {
	slog.Warn(redact(msg), sanitizeArgs(attrs)...)
}

// Error writes an error log entry.
func Error(msg string, attrs ...any) {
	slog.Error(redact(msg), sanitizeArgs(attrs)...)
}

func sanitizeArgs(attrs []any) []any {
	if len(attrs) == 0 {
		return attrs
	}

	out := make([]any, len(attrs))
	copy(out, attrs)

	for i, v := range out {
		switch vv := v.(type) {
		case string:
			out[i] = redact(vv)
		case error:
			out[i] = redact(vv.Error())
		}
	}

	return out
}

func redact(s string) string {
	return tokenPattern.ReplaceAllString(s, "[REDACTED]")
}
