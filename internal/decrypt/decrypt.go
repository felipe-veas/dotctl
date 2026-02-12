package decrypt

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Tool identifies a supported decryption backend.
type Tool string

const (
	ToolSOPS Tool = "sops"
	ToolAGE  Tool = "age"
)

var (
	lookPath = exec.LookPath
	runTool  = runToolOutput
)

// DetectTool returns the preferred decryption tool available on PATH.
// Preference order: sops, then age.
func DetectTool() (Tool, error) {
	if _, err := lookPath(string(ToolSOPS)); err == nil {
		return ToolSOPS, nil
	}
	if _, err := lookPath(string(ToolAGE)); err == nil {
		return ToolAGE, nil
	}
	return "", fmt.Errorf("decrypt entries require sops or age in PATH%s", installHint())
}

// DecryptFile decrypts sourcePath and returns plaintext bytes and the tool used.
func DecryptFile(sourcePath string) ([]byte, Tool, error) {
	tool, err := DetectTool()
	if err != nil {
		return nil, "", err
	}

	var out []byte
	switch tool {
	case ToolSOPS:
		out, err = runTool(string(tool), "--decrypt", sourcePath)
	case ToolAGE:
		out, err = runTool(string(tool), "--decrypt", "--output", "-", sourcePath)
	default:
		return nil, "", fmt.Errorf("unsupported decrypt tool: %s", tool)
	}
	if err != nil {
		return nil, "", fmt.Errorf("decrypting %q with %s: %w", sourcePath, tool, err)
	}

	return out, tool, nil
}

func runToolOutput(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.Output()
	if err == nil {
		return out, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		stderr := strings.TrimSpace(string(exitErr.Stderr))
		if stderr != "" {
			return nil, errors.New(stderr)
		}
	}

	return nil, err
}

func installHint() string {
	switch runtime.GOOS {
	case "darwin":
		return " (install with: brew install sops age)"
	case "linux":
		return " (install with your package manager, e.g. apt install sops age)"
	default:
		return ""
	}
}
