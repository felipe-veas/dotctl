package manifest

import (
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

// Context holds the resolved runtime context used for manifest condition evaluation.
type Context struct {
	OS       string
	Arch     string
	Hostname string
	Home     string
}

// RuntimeContext builds a Context from the current system state.
func RuntimeContext() Context {
	hostname, _ := os.Hostname()
	home, _ := os.UserHomeDir()

	return Context{
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		Hostname: hostname,
		Home:     home,
	}
}

// Vars returns template variables available in manifest targets.
func (c Context) Vars() map[string]string {
	return map[string]string{
		"home":     c.Home,
		"os":       c.OS,
		"arch":     c.Arch,
		"hostname": c.Hostname,
	}
}

// Action represents a resolved file action to execute.
type Action struct {
	Source     string // relative path in repo
	Target     string // absolute resolved target path
	Mode       string // "symlink" or "copy"
	Backup     bool   // whether to backup existing file
	SkipReason string // non-empty if skipped (for dry-run reporting)
}

// Resolve filters manifest entries by the current context and resolves targets.
// It returns a list of actions to apply and a list of skipped entries (for reporting).
func Resolve(m *Manifest, ctx Context) (actions []Action, skipped []Action, err error) {
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

		// Resolve target path
		resolvedTarget, resolveErr := ResolveTarget(f.Target, vars)
		if resolveErr != nil {
			return nil, nil, resolveErr
		}

		actions = append(actions, Action{
			Source: source,
			Target: resolvedTarget,
			Mode:   f.LinkMode(),
			Backup: f.ShouldBackup(),
		})
	}

	return actions, skipped, nil
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
