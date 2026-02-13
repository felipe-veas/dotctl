package secrets

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"filippo.io/age"
)

// GenerateIdentity creates a new age X25519 identity and writes it to path with 0600 permissions.
func GenerateIdentity(path string) (*Identity, error) {
	id, err := age.GenerateX25519Identity()
	if err != nil {
		return nil, fmt.Errorf("generating age identity: %w", err)
	}

	content := fmt.Sprintf("# created: %s\n# public key: %s\n%s\n",
		nowFunc().Format("2006-01-02T15:04:05-07:00"),
		id.Recipient().String(),
		id.String(),
	)

	dir := dirOf(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("creating identity directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return nil, fmt.Errorf("writing identity file: %w", err)
	}

	return &Identity{
		PrivatePath: path,
		PublicKey:   id.Recipient().String(),
		Recipient:   id.Recipient(),
		identity:    id,
	}, nil
}

// EncryptBytes encrypts plaintext for the given recipient public key string.
func EncryptBytes(plaintext []byte, recipientKey string) ([]byte, error) {
	recipient, err := age.ParseX25519Recipient(recipientKey)
	if err != nil {
		return nil, fmt.Errorf("parsing recipient key: %w", err)
	}

	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, recipient)
	if err != nil {
		return nil, fmt.Errorf("creating age writer: %w", err)
	}

	if _, err := w.Write(plaintext); err != nil {
		return nil, fmt.Errorf("writing plaintext: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("finalizing encryption: %w", err)
	}

	return buf.Bytes(), nil
}

// DecryptBytes decrypts ciphertext using the provided identity.
func DecryptBytes(ciphertext []byte, id *Identity) ([]byte, error) {
	if id.identity == nil {
		return nil, fmt.Errorf("identity has no private key loaded")
	}

	r, err := age.Decrypt(bytes.NewReader(ciphertext), id.identity)
	if err != nil {
		return nil, fmt.Errorf("decrypting: %w", err)
	}

	plaintext, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading decrypted data: %w", err)
	}

	return plaintext, nil
}

// EncryptFile reads a plaintext file and returns encrypted bytes.
func EncryptFile(path, recipientKey string) ([]byte, error) {
	plaintext, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file %q: %w", path, err)
	}
	return EncryptBytes(plaintext, recipientKey)
}

// DecryptFileWithIdentity reads an encrypted file and returns plaintext bytes.
func DecryptFileWithIdentity(path string, id *Identity) ([]byte, error) {
	ciphertext, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file %q: %w", path, err)
	}
	return DecryptBytes(ciphertext, id)
}
