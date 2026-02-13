package manifest

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/felipe-veas/dotctl/internal/profile"
)

// Action represents a resolved file action to execute.
type Action struct {
	Source     string // relative path in repo
	Target     string // absolute resolved target path
	Mode       string // "symlink" or "copy"
	Decrypt    bool   // whether source must be decrypted before copy
	Backup     bool   // whether to backup existing file
	SkipReason string // non-empty if skipped (for dry-run reporting)
}

// Resolve filters manifest entries by the current context and resolves targets.
// It returns a list of actions to apply and a list of skipped entries (for reporting).
func Resolve(m *Manifest, ctx profile.Context, repoRoot string) (actions []Action, skipped []Action, err error) {
	vars := MergeVars(m.Vars, ctx.Vars())

	for _, f := range m.Files {
		source, sourceErr := normalizeSourcePath(f.Source)
		if sourceErr != nil {
			return nil, nil, sourceErr
		}

		if pattern, ignored := matchedIgnorePattern(source, m.Ignore); ignored {
			skipped = append(skipped, Action{
				Source:     source,
				Target:     f.Target,
				SkipReason: "ignored by pattern " + pattern,
			})
			continue
		}

		// Evaluate conditions
		if !f.When.OS.Matches(ctx.OS) {
			skipped = append(skipped, Action{
				Source:     source,
				Target:     f.Target,
				SkipReason: "os: " + ctx.OS + " not in " + sliceStr(f.When.OS),
			})
			continue
		}

		if !f.When.Profile.Matches(ctx.Profile) {
			skipped = append(skipped, Action{
				Source:     source,
				Target:     f.Target,
				SkipReason: "profile: " + ctx.Profile + " not in " + sliceStr(f.When.Profile),
			})
			continue
		}

		// Resolve target path
		resolvedTarget, resolveErr := ResolveTarget(f.Target, vars)
		if resolveErr != nil {
			return nil, nil, resolveErr
		}

		actions = append(actions, Action{
			Source:  source,
			Target:  resolvedTarget,
			Mode:    f.LinkMode(),
			Decrypt: f.Decrypt,
			Backup:  f.ShouldBackup(),
		})
	}

	return actions, skipped, nil
}

// ResolveHooks filters hooks by the current context.
func ResolveHooks(hooks []Hook, ctx profile.Context) []Hook {
	var result []Hook
	for _, h := range hooks {
		if !h.When.OS.Matches(ctx.OS) {
			continue
		}
		if !h.When.Profile.Matches(ctx.Profile) {
			continue
		}
		result = append(result, h)
	}
	return result
}

func sliceStr(s StringOrSlice) string {
	if len(s) == 0 {
		return "(any)"
	}
	result := "["
	for i, v := range s {
		if i > 0 {
			result += ", "
		}
		result += v
	}
	return result + "]"
}

func matchedIgnorePattern(source string, patterns []string) (string, bool) {
	src := filepath.ToSlash(strings.TrimSpace(source))
	src = strings.TrimPrefix(src, "./")
	base := path.Base(src)

	for _, rawPattern := range patterns {
		pattern := filepath.ToSlash(strings.TrimSpace(rawPattern))
		if pattern == "" {
			continue
		}

		if ok, _ := path.Match(pattern, src); ok {
			return pattern, true
		}
		if ok, _ := path.Match(pattern, base); ok {
			return pattern, true
		}
	}

	return "", false
}
