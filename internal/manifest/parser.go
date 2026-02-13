package manifest

import (
	"fmt"
	"os"
	"path"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// Load reads and parses a manifest.yaml file.
func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	return Parse(data)
}

// Parse parses manifest YAML bytes.
func Parse(data []byte) (*Manifest, error) {
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest YAML (see line/column in error): %w", err)
	}

	if err := validate(&m); err != nil {
		return nil, err
	}

	return &m, nil
}

// validate checks the manifest for basic errors.
func validate(m *Manifest) error {
	seen := make(map[string]bool)
	for i := range m.Files {
		source, err := normalizeSourcePath(m.Files[i].Source)
		if err != nil {
			return fmt.Errorf("files[%d]: %w", i, err)
		}
		m.Files[i].Source = source

		f := m.Files[i]
		if f.Target == "" {
			return fmt.Errorf("files[%d]: target is required", i)
		}
		mode := f.LinkMode()
		if mode != "symlink" && mode != "copy" {
			return fmt.Errorf("files[%d]: invalid mode %q (must be 'symlink' or 'copy')", i, mode)
		}
		if f.Decrypt {
			if mode != "copy" {
				return fmt.Errorf("files[%d]: decrypt=true requires mode=copy", i)
			}
			if !hasEncryptedSuffix(f.Source) {
				return fmt.Errorf("files[%d]: decrypt=true requires encrypted source name containing '.enc.'", i)
			}
		}
		if seen[f.Target] {
			return fmt.Errorf("files[%d]: duplicate target %q", i, f.Target)
		}
		seen[f.Target] = true
	}
	return nil
}

func normalizeSourcePath(source string) (string, error) {
	trimmed := strings.TrimSpace(strings.ReplaceAll(source, "\\", "/"))
	if trimmed == "" {
		return "", fmt.Errorf("source is required")
	}

	normalized := path.Clean(trimmed)
	if normalized == "." {
		return "", fmt.Errorf("source is required")
	}
	if path.IsAbs(normalized) || isWindowsAbsolutePath(normalized) {
		return "", fmt.Errorf("source %q must be relative to repo root", source)
	}
	if normalized == ".." || strings.HasPrefix(normalized, "../") {
		return "", fmt.Errorf("source %q escapes repo root", source)
	}

	return normalized, nil
}

func isWindowsAbsolutePath(p string) bool {
	if len(p) < 3 {
		return false
	}
	drive := p[0]
	if (drive < 'A' || drive > 'Z') && (drive < 'a' || drive > 'z') {
		return false
	}
	return p[1] == ':' && p[2] == '/'
}

func hasEncryptedSuffix(source string) bool {
	base := strings.ToLower(strings.TrimSpace(path.Base(source)))
	return strings.Contains(base, ".enc.")
}

// ResolveTarget resolves template variables in a target path.
// vars is the merged map of manifest vars + built-in vars from profile.Context.
func ResolveTarget(target string, vars map[string]string) (string, error) {
	// Quick path: no templates
	if !strings.Contains(target, "{{") {
		return expandHome(target, vars["home"]), nil
	}

	tmpl, err := template.New("target").Option("missingkey=error").Parse(target)
	if err != nil {
		return "", fmt.Errorf("parsing target template %q: %w", target, err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("resolving target %q: %w", target, err)
	}

	return expandHome(buf.String(), vars["home"]), nil
}

// expandHome replaces a leading ~ with the home directory.
func expandHome(path, home string) string {
	if strings.HasPrefix(path, "~/") {
		return home + path[1:]
	}
	if path == "~" {
		return home
	}
	return path
}

// MergeVars merges manifest-defined vars with built-in context vars.
// Built-in vars take precedence for reserved names (home, os, arch, profile, hostname).
// Manifest vars fill in the rest (e.g. config_home).
func MergeVars(manifestVars, contextVars map[string]string) map[string]string {
	merged := make(map[string]string)

	// Manifest vars first
	for k, v := range manifestVars {
		// Resolve ~ in var values
		if home, ok := contextVars["home"]; ok {
			v = expandHome(v, home)
		}
		merged[k] = v
	}

	// Context vars override reserved names
	for k, v := range contextVars {
		merged[k] = v
	}

	return merged
}
