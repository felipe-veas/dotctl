package manifest

import (
	"testing"
)

func TestResolveFiltersByOS(t *testing.T) {
	m := &Manifest{
		Files: []FileEntry{
			{Source: "a", Target: "~/.a"},
			{Source: "b", Target: "~/.b", When: Condition{OS: StringOrSlice{"darwin"}}},
			{Source: "c", Target: "~/.c", When: Condition{OS: StringOrSlice{"linux"}}},
		},
	}

	ctx := Context{OS: "darwin", Arch: "arm64", Home: "/home/test"}
	actions, skipped, err := Resolve(m, ctx)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if len(actions) != 2 {
		t.Errorf("actions = %d, want 2 (a + b)", len(actions))
	}
	if len(skipped) != 1 {
		t.Errorf("skipped = %d, want 1 (c)", len(skipped))
	}
	if skipped[0].Source != "c" {
		t.Errorf("skipped source = %q, want %q", skipped[0].Source, "c")
	}
}

func TestResolveCombinedConditions(t *testing.T) {
	m := &Manifest{
		Files: []FileEntry{
			{
				Source: "a",
				Target: "~/.a",
				When: Condition{
					OS: StringOrSlice{"darwin"},
				},
			},
		},
	}

	ctx := Context{OS: "darwin", Home: "/home/test"}
	actions, _, err := Resolve(m, ctx)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(actions) != 1 {
		t.Errorf("both match: actions = %d, want 1", len(actions))
	}

	// OS doesn't match
	ctx.OS = "linux"
	actions, _, err = Resolve(m, ctx)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(actions) != 0 {
		t.Errorf("OS mismatch: actions = %d, want 0", len(actions))
	}
}

func TestResolveWithVars(t *testing.T) {
	m := &Manifest{
		Vars: map[string]string{
			"config_home": "~/.config",
		},
		Files: []FileEntry{
			{Source: "nvim", Target: "{{ .config_home }}/nvim"},
		},
	}

	ctx := Context{OS: "darwin", Home: "/Users/me"}
	actions, _, err := Resolve(m, ctx)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if len(actions) != 1 {
		t.Fatalf("actions = %d, want 1", len(actions))
	}
	if actions[0].Target != "/Users/me/.config/nvim" {
		t.Errorf("Target = %q, want %q", actions[0].Target, "/Users/me/.config/nvim")
	}
}

func TestResolveDefaultMode(t *testing.T) {
	m := &Manifest{
		Files: []FileEntry{
			{Source: "a", Target: "~/.a"},
			{Source: "b", Target: "~/.b", Mode: "copy"},
		},
	}

	ctx := Context{OS: "darwin", Home: "/home/test"}
	actions, _, err := Resolve(m, ctx)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if actions[0].Mode != "symlink" {
		t.Errorf("default mode = %q, want %q", actions[0].Mode, "symlink")
	}
	if actions[1].Mode != "copy" {
		t.Errorf("explicit mode = %q, want %q", actions[1].Mode, "copy")
	}
}

func TestResolveSkipsIgnoredSources(t *testing.T) {
	m := &Manifest{
		Ignore: []string{"*.key", ".env"},
		Files: []FileEntry{
			{Source: "configs/zsh/.zshrc", Target: "~/.zshrc"},
			{Source: "configs/keys/private.key", Target: "~/.private.key"},
			{Source: ".env", Target: "~/.env"},
		},
	}

	ctx := Context{OS: "darwin", Home: "/home/test"}
	actions, skipped, err := Resolve(m, ctx)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if len(actions) != 1 {
		t.Fatalf("actions = %d, want 1", len(actions))
	}
	if actions[0].Source != "configs/zsh/.zshrc" {
		t.Fatalf("unexpected action source: %s", actions[0].Source)
	}
	if len(skipped) != 2 {
		t.Fatalf("skipped = %d, want 2", len(skipped))
	}
}

func TestResolveIgnorePatternMatchesFullPath(t *testing.T) {
	m := &Manifest{
		Ignore: []string{"configs/private/*"},
		Files: []FileEntry{
			{Source: "configs/private/token.txt", Target: "~/.token"},
			{Source: "configs/public/config.toml", Target: "~/.config.toml"},
		},
	}

	ctx := Context{OS: "linux", Home: "/home/test"}
	actions, skipped, err := Resolve(m, ctx)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(actions) != 1 || actions[0].Source != "configs/public/config.toml" {
		t.Fatalf("unexpected actions: %+v", actions)
	}
	if len(skipped) != 1 {
		t.Fatalf("skipped = %d, want 1", len(skipped))
	}
	if skipped[0].SkipReason == "" {
		t.Fatal("expected skip reason for ignored entry")
	}
}

func TestResolveRejectsSourceTraversal(t *testing.T) {
	m := &Manifest{
		Files: []FileEntry{
			{Source: "../private/key", Target: "~/.ssh/id_rsa"},
		},
	}

	ctx := Context{OS: "linux", Home: "/home/test"}
	_, _, err := Resolve(m, ctx)
	if err == nil {
		t.Fatal("expected error for source path traversal")
	}
}

func TestResolveRejectsTargetOutsideHome(t *testing.T) {
	m := &Manifest{
		Files: []FileEntry{
			{Source: "configs/zsh/.zshrc", Target: "/etc/passwd"},
		},
	}

	ctx := Context{OS: "linux", Home: "/Users/test"}
	_, _, err := Resolve(m, ctx)
	if err == nil {
		t.Fatal("expected error for target outside home")
	}
}

func TestStringOrSliceMatches(t *testing.T) {
	tests := []struct {
		s     StringOrSlice
		value string
		want  bool
	}{
		{nil, "anything", true},             // empty = matches all
		{StringOrSlice{}, "anything", true}, // empty = matches all
		{StringOrSlice{"darwin"}, "darwin", true},
		{StringOrSlice{"darwin"}, "linux", false},
		{StringOrSlice{"darwin", "linux"}, "linux", true},
		{StringOrSlice{"darwin", "linux"}, "windows", false},
	}

	for _, tt := range tests {
		got := tt.s.Matches(tt.value)
		if got != tt.want {
			t.Errorf("StringOrSlice%v.Matches(%q) = %v, want %v", []string(tt.s), tt.value, got, tt.want)
		}
	}
}
