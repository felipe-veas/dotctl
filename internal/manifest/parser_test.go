package manifest

import (
	"testing"
)

func TestParseValid(t *testing.T) {
	data := []byte(`
version: 1
vars:
  config_home: "~/.config"
files:
  - source: configs/zsh/.zshrc
    target: ~/.zshrc
  - source: configs/nvim
    target: "{{ .config_home }}/nvim"
    mode: symlink
  - source: configs/brew/Brewfile
    target: "{{ .config_home }}/brew/Brewfile"
    when:
      os: darwin
  - source: configs/apt/packages.txt
    target: "{{ .config_home }}/apt/packages.txt"
    when:
      os: linux
  - source: configs/wezterm/wezterm.lua
    target: "{{ .config_home }}/wezterm/wezterm.lua"
    when:
      profile: [macstudio, laptop]
hooks:
  post_sync:
    - command: echo done
      when:
        os: darwin
ignore:
  - "*.token"
  - ".env"
`)

	m, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if m.Version != 1 {
		t.Errorf("Version = %d, want 1", m.Version)
	}
	if len(m.Files) != 5 {
		t.Errorf("Files count = %d, want 5", len(m.Files))
	}
	if len(m.Ignore) != 2 {
		t.Errorf("Ignore count = %d, want 2", len(m.Ignore))
	}
	if len(m.Hooks.PostSync) != 1 {
		t.Errorf("PostSync hooks = %d, want 1", len(m.Hooks.PostSync))
	}

	// Check condition on file[2]
	if len(m.Files[2].When.OS) != 1 || m.Files[2].When.OS[0] != "darwin" {
		t.Errorf("Files[2].When.OS = %v, want [darwin]", m.Files[2].When.OS)
	}

	// Check multi-profile condition on file[4]
	if len(m.Files[4].When.Profile) != 2 {
		t.Errorf("Files[4].When.Profile = %v, want [macstudio laptop]", m.Files[4].When.Profile)
	}
}

func TestParseMissingSource(t *testing.T) {
	data := []byte(`
version: 1
files:
  - target: ~/.zshrc
`)
	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for missing source")
	}
}

func TestParseRejectsAbsoluteSource(t *testing.T) {
	data := []byte(`
version: 1
files:
  - source: /etc/passwd
    target: ~/.zshrc
`)

	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for absolute source")
	}
}

func TestParseRejectsSourceTraversal(t *testing.T) {
	data := []byte(`
version: 1
files:
  - source: ../secrets/token.txt
    target: ~/.token
`)

	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for source path traversal")
	}
}

func TestParseRejectsWindowsAbsoluteSource(t *testing.T) {
	data := []byte(`
version: 1
files:
  - source: C:\Users\user\.ssh\id_rsa
    target: ~/.ssh/id_rsa
`)

	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for windows absolute source")
	}
}

func TestParseMissingTarget(t *testing.T) {
	data := []byte(`
version: 1
files:
  - source: configs/zsh/.zshrc
`)
	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for missing target")
	}
}

func TestParseInvalidMode(t *testing.T) {
	data := []byte(`
version: 1
files:
  - source: a
    target: b
    mode: hardlink
`)
	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

func TestParseDuplicateTarget(t *testing.T) {
	data := []byte(`
version: 1
files:
  - source: a
    target: ~/.zshrc
  - source: b
    target: ~/.zshrc
`)
	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for duplicate target")
	}
}

func TestParseDecryptRequiresCopyMode(t *testing.T) {
	data := []byte(`
version: 1
files:
  - source: configs/secrets/api.enc.yaml
    target: ~/.config/secrets/api.yaml
    mode: symlink
    decrypt: true
`)

	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for decrypt=true with non-copy mode")
	}
}

func TestParseDecryptRequiresEncryptedSourceName(t *testing.T) {
	data := []byte(`
version: 1
files:
  - source: configs/secrets/api.yaml
    target: ~/.config/secrets/api.yaml
    mode: copy
    decrypt: true
`)

	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for decrypt=true with non-encrypted source name")
	}
}

func TestResolveTarget(t *testing.T) {
	vars := map[string]string{
		"home":        "/Users/test",
		"config_home": "/Users/test/.config",
	}

	tests := []struct {
		input string
		want  string
	}{
		{"~/.zshrc", "/Users/test/.zshrc"},
		{"{{ .config_home }}/nvim", "/Users/test/.config/nvim"},
		{"/absolute/path", "/absolute/path"},
	}

	for _, tt := range tests {
		got, err := ResolveTarget(tt.input, vars)
		if err != nil {
			t.Errorf("ResolveTarget(%q): %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ResolveTarget(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestResolveTargetMissingVar(t *testing.T) {
	vars := map[string]string{"home": "/Users/test"}

	_, err := ResolveTarget("{{ .nonexistent }}/foo", vars)
	if err == nil {
		t.Fatal("expected error for missing template variable")
	}
}

func TestMergeVars(t *testing.T) {
	manifestVars := map[string]string{
		"config_home": "~/.config",
		"custom":      "value",
	}
	contextVars := map[string]string{
		"home":    "/Users/test",
		"os":      "darwin",
		"profile": "macstudio",
	}

	merged := MergeVars(manifestVars, contextVars)

	if merged["config_home"] != "/Users/test/.config" {
		t.Errorf("config_home = %q, want expanded path", merged["config_home"])
	}
	if merged["custom"] != "value" {
		t.Errorf("custom = %q, want %q", merged["custom"], "value")
	}
	if merged["home"] != "/Users/test" {
		t.Errorf("home = %q, want %q", merged["home"], "/Users/test")
	}
}

func TestExpandHome(t *testing.T) {
	tests := []struct {
		path, home, want string
	}{
		{"~/.config", "/home/user", "/home/user/.config"},
		{"~/", "/home/user", "/home/user/"},
		{"~", "/home/user", "/home/user"},
		{"/absolute", "/home/user", "/absolute"},
		{"relative", "/home/user", "relative"},
	}

	for _, tt := range tests {
		got := expandHome(tt.path, tt.home)
		if got != tt.want {
			t.Errorf("expandHome(%q, %q) = %q, want %q", tt.path, tt.home, got, tt.want)
		}
	}
}

func TestParseInvalidYAML(t *testing.T) {
	_, err := Parse([]byte(":\n  [bad"))
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}
