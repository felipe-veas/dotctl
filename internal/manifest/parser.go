package manifest

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
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
	if err := rejectDeprecatedFields(data); err != nil {
		return nil, err
	}

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
		if seen[f.Target] {
			return fmt.Errorf("files[%d]: duplicate target %q", i, f.Target)
		}
		seen[f.Target] = true
	}
	return nil
}

func rejectDeprecatedFields(data []byte) error {
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return fmt.Errorf("parsing manifest YAML (see line/column in error): %w", err)
	}

	if containsMappingKey(&node, "decrypt") {
		return fmt.Errorf("manifest uses deprecated field %q. dotctl no longer manages secret decryption. Store only non-sensitive dotfiles or decrypt secrets outside dotctl", "decrypt")
	}
	if containsMappingKey(&node, "profile") {
		return fmt.Errorf("manifest uses deprecated field %q. dotctl now manages a single configuration set. Remove profile filters from manifest.yaml", "when.profile")
	}
	if containsMappingKey(&node, "hooks") {
		return fmt.Errorf("manifest uses deprecated field %q. dotctl no longer executes commands from manifest.yaml. Run setup commands manually outside dotctl", "hooks")
	}

	return nil
}

func containsMappingKey(node *yaml.Node, key string) bool {
	if node == nil {
		return false
	}

	if node.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(node.Content); i += 2 {
			if node.Content[i].Value == key {
				return true
			}
			if containsMappingKey(node.Content[i+1], key) {
				return true
			}
		}
	}

	for _, child := range node.Content {
		if containsMappingKey(child, key) {
			return true
		}
	}

	return false
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

// ResolveTarget resolves template variables in a target path.
// vars is the merged map of manifest vars + built-in vars from Context.
func ResolveTarget(target string, vars map[string]string) (string, error) {
	home := strings.TrimSpace(vars["home"])
	if home == "" {
		return "", fmt.Errorf("resolving target %q: home variable is required", target)
	}

	resolved, err := resolveTargetTemplate(target, vars, home)
	if err != nil {
		return "", err
	}

	return validateResolvedTarget(target, resolved, home)
}

func resolveTargetTemplate(target string, vars map[string]string, home string) (string, error) {
	// Quick path: no templates
	if !strings.Contains(target, "{{") {
		return expandHome(target, home), nil
	}

	tmpl, err := template.New("target").Option("missingkey=error").Parse(target)
	if err != nil {
		return "", fmt.Errorf("parsing target template %q: %w", target, err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("resolving target %q: %w", target, err)
	}

	return expandHome(buf.String(), home), nil
}

func validateResolvedTarget(original, resolved, home string) (string, error) {
	trimmed := strings.TrimSpace(resolved)
	if trimmed == "" {
		return "", fmt.Errorf("resolving target %q: resolved target is empty", original)
	}

	cleanHome := filepath.Clean(strings.TrimSpace(home))
	if cleanHome == "" || cleanHome == "." {
		return "", fmt.Errorf("resolving target %q: home directory is required", original)
	}
	if !filepath.IsAbs(cleanHome) {
		return "", fmt.Errorf("resolving target %q: home directory %q must be absolute", original, home)
	}

	cleanTarget := filepath.Clean(trimmed)
	if !filepath.IsAbs(cleanTarget) {
		return "", fmt.Errorf("resolving target %q: resolved target %q must be absolute", original, trimmed)
	}

	rel, err := filepath.Rel(cleanHome, cleanTarget)
	if err != nil {
		return "", fmt.Errorf("resolving target %q: checking target containment: %w", original, err)
	}
	if rel == "." {
		return "", fmt.Errorf("resolving target %q: target %q must not equal home directory %q", original, cleanTarget, cleanHome)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("resolving target %q: target %q must stay within home directory %q", original, cleanTarget, cleanHome)
	}

	return cleanTarget, nil
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
// Built-in vars take precedence for reserved names (home, os, arch, hostname).
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
