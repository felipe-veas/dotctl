package cmd

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/felipe-veas/dotctl/internal/decrypt"
	"github.com/felipe-veas/dotctl/internal/manifest"
	"github.com/felipe-veas/dotctl/internal/output"
	"github.com/spf13/cobra"
)

type diffEntry struct {
	Source  string `json:"source"`
	Target  string `json:"target"`
	Mode    string `json:"mode"`
	Decrypt bool   `json:"decrypt,omitempty"`
	Status  string `json:"status"` // ok, changed, missing, drift, error
	Reason  string `json:"reason,omitempty"`
	Diff    string `json:"diff,omitempty"`
}

type diffResult struct {
	Profile  string      `json:"profile"`
	RepoName string      `json:"repo_name,omitempty"`
	RepoPath string      `json:"repo_path"`
	Entries  []diffEntry `json:"entries"`
	Summary  struct {
		Total   int `json:"total"`
		OK      int `json:"ok"`
		Changed int `json:"changed"`
		Missing int `json:"missing"`
		Drift   int `json:"drift"`
		Errors  int `json:"errors"`
	} `json:"summary"`
}

func newDiffCmd() *cobra.Command {
	var showDetails bool

	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Show differences between repo files and current local state",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDiff(showDetails)
		},
	}
	cmd.Flags().BoolVar(&showDetails, "details", false, "include unified diffs for changed files")
	return cmd
}

func runDiff(showDetails bool) error {
	out := output.New(flagJSON)

	cfg, _, err := resolveConfig()
	if err != nil {
		return err
	}

	state, err := resolveManifestState(cfg)
	if err != nil {
		return err
	}

	if _, decryptCount, decryptErr := detectDecryptToolForActions(state.Actions); decryptCount > 0 && decryptErr != nil {
		return decryptErr
	}

	result := diffResult{
		Profile:  cfg.Profile,
		RepoName: cfg.Repo.Name,
		RepoPath: cfg.Repo.Path,
		Entries:  make([]diffEntry, 0, len(state.Actions)),
	}

	for _, action := range state.Actions {
		sourcePath := filepath.Join(cfg.Repo.Path, action.Source)
		entry := diffAction(action, sourcePath, showDetails)
		result.Entries = append(result.Entries, entry)

		switch entry.Status {
		case "ok":
			result.Summary.OK++
		case "changed":
			result.Summary.Changed++
		case "missing":
			result.Summary.Missing++
		case "drift":
			result.Summary.Drift++
		default:
			result.Summary.Errors++
		}
	}
	result.Summary.Total = len(result.Entries)

	if out.IsJSON() {
		return out.JSON(result)
	}

	if result.RepoName != "" {
		out.Field("Repo name", result.RepoName)
	}
	out.Field("Repo path", result.RepoPath)
	out.Field("Profile", result.Profile)

	for _, entry := range result.Entries {
		switch entry.Status {
		case "ok":
			continue
		case "changed":
			out.Warn("%s → %s (%s)", entry.Source, entry.Target, entry.Reason)
		case "missing":
			out.Warn("%s → %s (missing: %s)", entry.Source, entry.Target, entry.Reason)
		case "drift":
			out.Warn("%s → %s (drift: %s)", entry.Source, entry.Target, entry.Reason)
		case "error":
			out.Error("%s → %s: %s", entry.Source, entry.Target, entry.Reason)
		}

		if showDetails && strings.TrimSpace(entry.Diff) != "" {
			out.Info("%s", strings.TrimSpace(entry.Diff))
		}
	}

	if result.Summary.Changed == 0 && result.Summary.Missing == 0 && result.Summary.Drift == 0 && result.Summary.Errors == 0 {
		out.Success("No differences found (%d/%d entries match).", result.Summary.OK, result.Summary.Total)
		return nil
	}

	out.Info("")
	out.Info("Summary: %d ok, %d changed, %d missing, %d drift, %d errors",
		result.Summary.OK,
		result.Summary.Changed,
		result.Summary.Missing,
		result.Summary.Drift,
		result.Summary.Errors,
	)
	return nil
}

func diffAction(action manifest.Action, sourcePath string, showDetails bool) diffEntry {
	entry := diffEntry{
		Source:  action.Source,
		Target:  action.Target,
		Mode:    action.Mode,
		Decrypt: action.Decrypt,
		Status:  "ok",
	}

	if _, err := os.Stat(sourcePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			entry.Status = "missing"
			entry.Reason = "source missing in repo"
			return entry
		}
		entry.Status = "error"
		entry.Reason = fmt.Sprintf("reading source: %v", err)
		return entry
	}

	switch action.Mode {
	case "symlink":
		return diffSymlink(entry, sourcePath)
	case "copy":
		return diffCopy(entry, action, sourcePath, showDetails)
	default:
		entry.Status = "error"
		entry.Reason = fmt.Sprintf("unsupported mode %q", action.Mode)
		return entry
	}
}

func diffSymlink(entry diffEntry, sourcePath string) diffEntry {
	info, err := os.Lstat(entry.Target)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			entry.Status = "missing"
			entry.Reason = "target missing"
			return entry
		}
		entry.Status = "error"
		entry.Reason = fmt.Sprintf("reading target: %v", err)
		return entry
	}

	if info.Mode()&os.ModeSymlink == 0 {
		entry.Status = "drift"
		entry.Reason = "target exists but is not a symlink"
		return entry
	}

	dest, err := os.Readlink(entry.Target)
	if err != nil {
		entry.Status = "error"
		entry.Reason = fmt.Sprintf("reading symlink: %v", err)
		return entry
	}

	if dest != sourcePath {
		entry.Status = "drift"
		entry.Reason = fmt.Sprintf("points to %s", dest)
		return entry
	}

	return entry
}

func diffCopy(entry diffEntry, action manifest.Action, sourcePath string, showDetails bool) diffEntry {
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		entry.Status = "error"
		entry.Reason = fmt.Sprintf("reading source: %v", err)
		return entry
	}

	if sourceInfo.IsDir() {
		return diffCopyDirectory(entry, sourcePath)
	}

	targetInfo, err := os.Stat(entry.Target)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			entry.Status = "missing"
			entry.Reason = "target missing"
			return entry
		}
		entry.Status = "error"
		entry.Reason = fmt.Sprintf("reading target: %v", err)
		return entry
	}
	if targetInfo.IsDir() {
		entry.Status = "drift"
		entry.Reason = "target is directory, expected file"
		return entry
	}

	sourceData, sourceLabel, readErr := readDiffSource(action, sourcePath)
	if readErr != nil {
		entry.Status = "error"
		entry.Reason = readErr.Error()
		return entry
	}
	targetData, err := os.ReadFile(entry.Target)
	if err != nil {
		entry.Status = "error"
		entry.Reason = fmt.Sprintf("reading target file: %v", err)
		return entry
	}

	if bytes.Equal(sourceData, targetData) {
		return entry
	}

	entry.Status = "changed"
	entry.Reason = "content differs"
	if showDetails {
		if d, diffErr := unifiedDiff(sourceData, targetData, sourceLabel, entry.Target); diffErr == nil {
			entry.Diff = d
		}
	}
	return entry
}

func diffCopyDirectory(entry diffEntry, sourceDir string) diffEntry {
	targetInfo, err := os.Stat(entry.Target)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			entry.Status = "missing"
			entry.Reason = "target directory missing"
			return entry
		}
		entry.Status = "error"
		entry.Reason = fmt.Sprintf("reading target directory: %v", err)
		return entry
	}
	if !targetInfo.IsDir() {
		entry.Status = "drift"
		entry.Reason = "target is file, expected directory"
		return entry
	}

	sourceMap, err := directoryDigest(sourceDir)
	if err != nil {
		entry.Status = "error"
		entry.Reason = fmt.Sprintf("walking source dir: %v", err)
		return entry
	}
	targetMap, err := directoryDigest(entry.Target)
	if err != nil {
		entry.Status = "error"
		entry.Reason = fmt.Sprintf("walking target dir: %v", err)
		return entry
	}

	changed := make([]string, 0)
	for rel, srcDigest := range sourceMap {
		dstDigest, ok := targetMap[rel]
		if !ok || dstDigest != srcDigest {
			changed = append(changed, rel)
		}
	}
	for rel := range targetMap {
		if _, ok := sourceMap[rel]; !ok {
			changed = append(changed, rel)
		}
	}

	if len(changed) == 0 {
		return entry
	}

	sort.Strings(changed)
	entry.Status = "changed"
	entry.Reason = fmt.Sprintf("directory content differs (%s)", previewItems(changed, 5))
	return entry
}

func readDiffSource(action manifest.Action, sourcePath string) ([]byte, string, error) {
	if action.Decrypt {
		data, _, err := decrypt.DecryptFile(sourcePath)
		if err != nil {
			return nil, "", fmt.Errorf("decrypting source: %w", err)
		}
		return data, sourcePath + " (decrypted)", nil
	}

	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, "", fmt.Errorf("reading source file: %w", err)
	}
	return data, sourcePath, nil
}

func directoryDigest(root string) (map[string]string, error) {
	result := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		result[filepath.ToSlash(rel)] = fmt.Sprintf("%d:%x", len(data), sum)
		return nil
	})
	return result, err
}

func unifiedDiff(sourceData, targetData []byte, sourceLabel, targetLabel string) (string, error) {
	sourceTmp, err := os.CreateTemp("", "dotctl-diff-source-*")
	if err != nil {
		return "", err
	}
	_ = sourceTmp.Close()
	defer func() { _ = os.Remove(sourceTmp.Name()) }()
	if err := os.WriteFile(sourceTmp.Name(), sourceData, 0o600); err != nil {
		return "", err
	}

	targetTmp, err := os.CreateTemp("", "dotctl-diff-target-*")
	if err != nil {
		return "", err
	}
	_ = targetTmp.Close()
	defer func() { _ = os.Remove(targetTmp.Name()) }()
	if err := os.WriteFile(targetTmp.Name(), targetData, 0o600); err != nil {
		return "", err
	}

	args := []string{
		"-u",
		"--label", sourceLabel,
		"--label", targetLabel,
		sourceTmp.Name(),
		targetTmp.Name(),
	}
	cmd := exec.Command("diff", args...)
	combined, err := cmd.CombinedOutput()
	if err == nil {
		return "", nil
	}

	exitErr := &exec.ExitError{}
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return string(combined), nil
	}
	return "", err
}

func previewItems(items []string, limit int) string {
	if len(items) <= limit {
		return strings.Join(items, ", ")
	}
	return fmt.Sprintf("%s (+%d more)", strings.Join(items[:limit], ", "), len(items)-limit)
}
