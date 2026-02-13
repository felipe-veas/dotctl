package secrets

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Init generates a new age identity or imports an existing one.
// It writes the recipient file to the repo root and ensures .gitignore is updated.
func Init(repoRoot string, opts InitOptions) (*Identity, error) {
	idPath := opts.IdentityPath
	if idPath == "" {
		idPath = DefaultIdentityPath()
	}

	// Check for existing identity.
	if !opts.Force && IdentityExists(idPath) {
		return nil, fmt.Errorf("identity already exists at %q (use --force to overwrite)", idPath)
	}

	var id *Identity
	var err error

	if opts.ImportPath != "" {
		id, err = importIdentity(opts.ImportPath, idPath)
	} else {
		id, err = GenerateIdentity(idPath)
	}
	if err != nil {
		return nil, err
	}

	// Write recipient file to repo.
	if err := WriteRecipientFile(repoRoot, id.PublicKey); err != nil {
		return nil, fmt.Errorf("writing recipient file: %w", err)
	}

	// Ensure .gitignore contains the identity filename.
	if err := ensureGitignore(repoRoot, DefaultIdentityFile); err != nil {
		return nil, fmt.Errorf("updating .gitignore: %w", err)
	}

	return id, nil
}

// importIdentity copies an existing identity file to the target path.
func importIdentity(srcPath, dstPath string) (*Identity, error) {
	// Validate the source is a valid identity.
	id, err := FindIdentity(srcPath)
	if err != nil {
		return nil, fmt.Errorf("validating import file: %w", err)
	}

	// Read source content.
	content, err := os.ReadFile(srcPath)
	if err != nil {
		return nil, fmt.Errorf("reading import file: %w", err)
	}

	// Create parent dir and write with restricted permissions.
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o700); err != nil {
		return nil, fmt.Errorf("creating identity directory: %w", err)
	}
	if err := os.WriteFile(dstPath, content, 0o600); err != nil {
		return nil, fmt.Errorf("writing identity file: %w", err)
	}

	id.PrivatePath = dstPath
	return id, nil
}

// ensureGitignore adds pattern to .gitignore in repoRoot if not already present.
func ensureGitignore(repoRoot, pattern string) error {
	gitignorePath := filepath.Join(repoRoot, ".gitignore")

	// Read existing .gitignore if present.
	var lines []string
	if f, err := os.Open(gitignorePath); err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) == pattern {
				_ = f.Close()
				return nil // already present
			}
			lines = append(lines, line)
		}
		_ = f.Close()
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("reading .gitignore: %w", err)
		}
	}

	// Append the pattern.
	f, err := os.OpenFile(gitignorePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("opening .gitignore: %w", err)
	}
	defer func() { _ = f.Close() }()

	// Add a newline before if file doesn't end with one.
	if len(lines) > 0 {
		info, _ := f.Stat()
		if info.Size() > 0 {
			// Seek to check last byte.
			if _, err := f.Seek(-1, io.SeekEnd); err == nil {
				buf := make([]byte, 1)
				if _, err := f.Read(buf); err == nil && buf[0] != '\n' {
					_, _ = fmt.Fprintln(f)
				}
			}
		}
	}

	_, _ = fmt.Fprintf(f, "\n# dotctl secrets\n%s\n", pattern)
	return nil
}
