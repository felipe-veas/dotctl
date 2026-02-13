package secrets

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"
	"github.com/felipe-veas/dotctl/internal/platform"
)

// DefaultIdentityPath returns the default path for the age identity file.
func DefaultIdentityPath() string {
	return filepath.Join(platform.ConfigDir(), DefaultIdentityFile)
}

// FindIdentity locates and parses the age identity file.
// If path is empty, the default location is used.
func FindIdentity(path string) (*Identity, error) {
	if path == "" {
		path = DefaultIdentityPath()
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening identity file: %w", err)
	}
	defer func() { _ = f.Close() }()

	identities, err := age.ParseIdentities(f)
	if err != nil {
		return nil, fmt.Errorf("parsing identity file %q: %w", path, err)
	}

	if len(identities) == 0 {
		return nil, fmt.Errorf("no identities found in %q", path)
	}

	x25519, ok := identities[0].(*age.X25519Identity)
	if !ok {
		return nil, fmt.Errorf("unsupported identity type in %q (expected X25519)", path)
	}

	return &Identity{
		PrivatePath: path,
		PublicKey:   x25519.Recipient().String(),
		Recipient:   x25519.Recipient(),
		identity:    x25519,
	}, nil
}

// FindRecipient reads the public key from .age-recipient.txt in the repo root.
func FindRecipient(repoRoot string) (string, error) {
	path := filepath.Join(repoRoot, DefaultRecipientFile)

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading recipient file: %w", err)
	}

	// Parse the file: skip comments and blank lines, take the first recipient.
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Validate it's a valid age recipient.
		if _, err := age.ParseX25519Recipient(line); err != nil {
			return "", fmt.Errorf("invalid recipient in %q: %w", path, err)
		}
		return line, nil
	}

	return "", fmt.Errorf("no recipient found in %q", path)
}

// WriteRecipientFile writes the public key to .age-recipient.txt in the repo root.
func WriteRecipientFile(repoRoot, publicKey string) error {
	path := filepath.Join(repoRoot, DefaultRecipientFile)
	content := fmt.Sprintf("# age public key for dotctl secrets\n%s\n", publicKey)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing recipient file: %w", err)
	}
	return nil
}

// IdentityExists returns true if an identity file exists at the given or default path.
func IdentityExists(path string) bool {
	if path == "" {
		path = DefaultIdentityPath()
	}
	_, err := os.Stat(path)
	return err == nil
}

// RecipientExists returns true if .age-recipient.txt exists in the repo root.
func RecipientExists(repoRoot string) bool {
	path := filepath.Join(repoRoot, DefaultRecipientFile)
	_, err := os.Stat(path)
	return err == nil
}
