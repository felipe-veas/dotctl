package manifest

import "fmt"

// Manifest represents the top-level manifest.yaml structure.
type Manifest struct {
	Version int               `yaml:"version"`
	Vars    map[string]string `yaml:"vars"`
	Files   []FileEntry       `yaml:"files"`
	Ignore  []string          `yaml:"ignore"`
	Hooks   HookSet           `yaml:"hooks"`
}

// FileEntry represents a single file mapping in the manifest.
type FileEntry struct {
	Source  string    `yaml:"source"`
	Target string    `yaml:"target"`
	Mode   string    `yaml:"mode"`    // "symlink" (default) or "copy"
	When   Condition `yaml:"when"`
	Decrypt bool     `yaml:"decrypt"`
	Backup  *bool    `yaml:"backup"` // nil = default true
}

// ShouldBackup returns whether this entry should create a backup before overwriting.
func (f FileEntry) ShouldBackup() bool {
	if f.Backup == nil {
		return true
	}
	return *f.Backup
}

// LinkMode returns the resolved mode, defaulting to "symlink".
func (f FileEntry) LinkMode() string {
	if f.Mode == "" {
		return "symlink"
	}
	return f.Mode
}

// Condition represents when-filters for OS and profile.
type Condition struct {
	OS      StringOrSlice `yaml:"os"`
	Profile StringOrSlice `yaml:"profile"`
}

// HookSet contains the different hook phases.
type HookSet struct {
	PreSync   []Hook `yaml:"pre_sync"`
	PostSync  []Hook `yaml:"post_sync"`
	Bootstrap []Hook `yaml:"bootstrap"`
}

// Hook represents a command to run at a specific phase.
type Hook struct {
	Command     string    `yaml:"command"`
	Description string    `yaml:"description"`
	When        Condition `yaml:"when"`
}

// StringOrSlice allows YAML values like "darwin" or ["darwin", "linux"].
type StringOrSlice []string

// UnmarshalYAML implements custom YAML unmarshaling for StringOrSlice.
func (s *StringOrSlice) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Try string first
	var single string
	if err := unmarshal(&single); err == nil {
		*s = StringOrSlice{single}
		return nil
	}

	// Try slice
	var multi []string
	if err := unmarshal(&multi); err == nil {
		*s = StringOrSlice(multi)
		return nil
	}

	return fmt.Errorf("expected string or list of strings")
}

// Matches returns true if the value is in the slice, or the slice is empty (no filter).
func (s StringOrSlice) Matches(value string) bool {
	if len(s) == 0 {
		return true
	}
	for _, v := range s {
		if v == value {
			return true
		}
	}
	return false
}
