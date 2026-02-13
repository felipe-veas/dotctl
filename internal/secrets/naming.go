package secrets

import (
	"path/filepath"
	"strings"
)

const encMarker = ".enc."

// EncryptedName converts a plaintext filename to its encrypted counterpart.
//
// Examples:
//
//	config.yaml     → config.enc.yaml
//	.env            → .env.enc
//	.env.local      → .env.enc.local
//	api.key         → api.enc.key
//	data.tar.gz     → data.tar.enc.gz
func EncryptedName(name string) string {
	if IsEncryptedName(name) {
		return name
	}

	dir := filepath.Dir(name)
	base := filepath.Base(name)

	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)

	var result string
	if ext == "" {
		// No extension: append .enc (e.g., Makefile → Makefile.enc)
		result = base + ".enc"
	} else if stem == "" || stem == "." {
		// Dotfile with no stem beyond the dot (e.g., .env → .env.enc)
		result = base + ".enc"
	} else {
		// Normal file: insert .enc before last extension (e.g., config.yaml → config.enc.yaml)
		result = stem + ".enc" + ext
	}

	if dir == "." {
		return result
	}
	return filepath.Join(dir, result)
}

// DecryptedName converts an encrypted filename back to its plaintext counterpart.
//
// Examples:
//
//	config.enc.yaml → config.yaml
//	.env.enc        → .env
//	.env.enc.local  → .env.local
//	api.enc.key     → api.key
func DecryptedName(name string) string {
	if !IsEncryptedName(name) {
		return name
	}

	dir := filepath.Dir(name)
	base := filepath.Base(name)

	// Handle trailing .enc (e.g., .env.enc, Makefile.enc)
	if strings.HasSuffix(base, ".enc") {
		result := strings.TrimSuffix(base, ".enc")
		if dir == "." {
			return result
		}
		return filepath.Join(dir, result)
	}

	// Handle .enc. in the middle (e.g., config.enc.yaml)
	result := strings.Replace(base, ".enc.", ".", 1)
	if dir == "." {
		return result
	}
	return filepath.Join(dir, result)
}

// IsEncryptedName returns true if the filename contains the .enc marker.
func IsEncryptedName(name string) bool {
	base := filepath.Base(name)
	return strings.Contains(base, encMarker) || strings.HasSuffix(base, ".enc")
}

// IsSensitiveName returns true if the filename matches known sensitive patterns.
func IsSensitiveName(name string) bool {
	base := filepath.Base(name)
	if IsEncryptedName(name) {
		return false
	}
	for _, pattern := range SensitivePatterns {
		if matched, _ := filepath.Match(pattern, base); matched {
			return true
		}
	}
	return false
}
