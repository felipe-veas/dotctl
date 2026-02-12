package cmd

import "testing"

func TestIsSensitiveTrackedPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: ".env", want: true},
		{path: "configs/.env.local", want: true},
		{path: "keys/private.pem", want: true},
		{path: "keys/service.key", want: true},
		{path: ".ssh/id_rsa", want: true},
		{path: "files/.ssh/id_ed25519", want: true},
		{path: "configs/zsh/.zshrc", want: false},
		{path: "README.md", want: false},
	}

	for _, tc := range tests {
		if got := isSensitiveTrackedPath(tc.path); got != tc.want {
			t.Errorf("isSensitiveTrackedPath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestSensitiveTrackedFilesWarning(t *testing.T) {
	msg := sensitiveTrackedFilesWarning([]string{".env", "secret.key"})
	if msg == "" {
		t.Fatal("expected non-empty warning")
	}
}
