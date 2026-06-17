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
	// Check condition on file[2]
	if len(m.Files[2].When.OS) != 1 || m.Files[2].When.OS[0] != "darwin" {
		t.Errorf("Files[2].When.OS = %v, want [darwin]", m.Files[2].When.OS)
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

func TestParseRejectsNestedSourceTraversal(t *testing.T) {
	data := []byte(`
version: 1
files:
  - source: configs/../../secret
    target: ~/.token
`)

	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for nested source path traversal")
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

func TestParseRejectsDeprecatedDecryptField(t *testing.T) {
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
		t.Fatal("expected error for deprecated decrypt field")
	}
}

func TestParseRejectsDeprecatedProfileCondition(t *testing.T) {
	data := []byte(`
version: 1
files:
  - source: configs/zsh/.zshrc
    target: ~/.zshrc
    when:
      profile: laptop
`)

	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for deprecated profile condition")
	}
}

func TestParseRejectsDeprecatedHooks(t *testing.T) {
	data := []byte(`
version: 1
files:
  - source: configs/zsh/.zshrc
    target: ~/.zshrc
hooks:
  post_sync:
    - command: echo done
`)

	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for deprecated hooks field")
	}
}

func TestResolveTarget(t *testing.T) {
	vars := map[string]string{
		"home":        "/Users/test",
		"config_home": "/Users/test/.config",
	}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "home expansion", input: "~/.zshrc", want: "/Users/test/.zshrc"},
		{name: "template expansion", input: "{{ .config_home }}/nvim", want: "/Users/test/.config/nvim"},
		{name: "absolute under home", input: "/Users/test/.config/nvim", want: "/Users/test/.config/nvim"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveTarget(tt.input, vars)
			if err != nil {
				t.Fatalf("ResolveTarget(%q): %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("ResolveTarget(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveTargetRejectsUnsafePaths(t *testing.T) {
	tests := []struct {
		name  string
		input string
		vars  map[string]string
	}{
		{name: "outside home absolute", input: "/absolute/path", vars: map[string]string{"home": "/Users/test"}},
		{name: "sibling home path", input: "/Users/test2/.config/nvim", vars: map[string]string{"home": "/Users/test"}},
		{name: "relative path", input: "relative/path", vars: map[string]string{"home": "/Users/test"}},
		{name: "home root", input: "~", vars: map[string]string{"home": "/Users/test"}},
		{name: "parent traversal with home", input: "~/../outside", vars: map[string]string{"home": "/Users/test"}},
		{name: "parent traversal with template", input: "{{ .home }}/../outside", vars: map[string]string{"home": "/Users/test"}},
		{name: "missing home", input: "~/.zshrc", vars: map[string]string{}},
		{name: "empty home", input: "~/.zshrc", vars: map[string]string{"home": "   "}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveTarget(tt.input, tt.vars)
			if err == nil {
				t.Fatalf("expected error for %q", tt.input)
			}
		})
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
		"home": "/Users/test",
		"os":   "darwin",
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
