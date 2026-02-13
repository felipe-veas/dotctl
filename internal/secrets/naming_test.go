package secrets

import "testing"

func TestEncryptedName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"config.yaml", "config.enc.yaml"},
		{".env", ".env.enc"},
		{".env.local", ".env.enc.local"},
		{"api.key", "api.enc.key"},
		{"data.tar.gz", "data.tar.enc.gz"},
		{"Makefile", "Makefile.enc"},
		{"configs/app/config.yaml", "configs/app/config.enc.yaml"},
		{"configs/env/.env", "configs/env/.env.enc"},
		// Already encrypted: no change.
		{"config.enc.yaml", "config.enc.yaml"},
		{".env.enc", ".env.enc"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := EncryptedName(tt.input)
			if got != tt.want {
				t.Errorf("EncryptedName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDecryptedName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"config.enc.yaml", "config.yaml"},
		{".env.enc", ".env"},
		{".env.enc.local", ".env.local"},
		{"api.enc.key", "api.key"},
		{"Makefile.enc", "Makefile"},
		{"configs/app/config.enc.yaml", "configs/app/config.yaml"},
		{"configs/env/.env.enc", "configs/env/.env"},
		// Not encrypted: no change.
		{"config.yaml", "config.yaml"},
		{".env", ".env"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := DecryptedName(tt.input)
			if got != tt.want {
				t.Errorf("DecryptedName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRoundtrip(t *testing.T) {
	names := []string{"config.yaml", ".env", ".env.local", "api.key", "Makefile"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			enc := EncryptedName(name)
			dec := DecryptedName(enc)
			if dec != name {
				t.Errorf("roundtrip failed: %q -> %q -> %q", name, enc, dec)
			}
		})
	}
}

func TestIsEncryptedName(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"config.enc.yaml", true},
		{".env.enc", true},
		{".env.enc.local", true},
		{"config.yaml", false},
		{".env", false},
		{"Makefile", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IsEncryptedName(tt.input)
			if got != tt.want {
				t.Errorf("IsEncryptedName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsSensitiveName(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{".env", true},
		{".env.local", true},
		{"api.key", true},
		{"cert.pem", true},
		{"config.yaml", false},
		{".env.enc", false},       // encrypted = not sensitive
		{"api.enc.key", false},    // encrypted = not sensitive
		{"README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IsSensitiveName(tt.input)
			if got != tt.want {
				t.Errorf("IsSensitiveName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
