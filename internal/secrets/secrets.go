package secrets

import "filippo.io/age"

const (
	// DefaultIdentityFile is the filename for the age private key.
	DefaultIdentityFile = "age-identity.txt"
	// DefaultRecipientFile is the filename for the age public key in the repo.
	DefaultRecipientFile = ".age-recipient.txt"
)

// SensitivePatterns lists filename patterns that should be encrypted.
var SensitivePatterns = []string{
	".env",
	".env.*",
	"*.key",
	"*.pem",
	"*.secret",
	"*.credentials",
}

// Identity represents a resolved age key pair.
type Identity struct {
	PrivatePath string // absolute path to the identity file
	PublicKey   string // age1... public key string
	Recipient  *age.X25519Recipient
	identity   *age.X25519Identity
}

// InitOptions configures Init behavior.
type InitOptions struct {
	IdentityPath string // override default identity file location
	ImportPath   string // path to existing identity file to import
	Force        bool   // overwrite existing identity
}

// EncryptOptions configures Encrypt behavior.
type EncryptOptions struct {
	RecipientKey string // override recipient public key
	Keep         bool   // keep original plaintext file after encrypting
}

// DecryptOptions configures Decrypt behavior.
type DecryptOptions struct {
	IdentityPath string // override identity file location
	Keep         bool   // keep encrypted file after decrypting
	Stdout       bool   // return bytes instead of writing to file
}

// RotateOptions configures Rotate behavior.
type RotateOptions struct {
	IdentityPath string // override new identity file location
}

// RotateResult describes the outcome of a key rotation.
type RotateResult struct {
	NewIdentity   *Identity
	BackupKeyPath string   // path where old key was backed up
	ReEncrypted   []string // files that were re-encrypted
}

// FileStatus describes the protection state of a single file.
type FileStatus struct {
	Path      string // relative path in repo
	Encrypted bool   // true if the file is encrypted
}

// Status describes the overall secrets health of a repository.
type Status struct {
	Identity         *Identity
	RecipientFile    string       // path to .age-recipient.txt in repo (empty if missing)
	EncryptedFiles   []FileStatus // files with .enc. in the repo
	UnprotectedFiles []FileStatus // sensitive files that should be encrypted
}
