package decrypt

import (
	"errors"
	"testing"
)

func TestDetectToolPrefersSOPS(t *testing.T) {
	origLookPath := lookPath
	t.Cleanup(func() { lookPath = origLookPath })

	lookPath = func(file string) (string, error) {
		switch file {
		case "sops":
			return "/tmp/sops", nil
		default:
			return "", errors.New("not found")
		}
	}

	tool, err := DetectTool()
	if err != nil {
		t.Fatalf("DetectTool() error = %v", err)
	}
	if tool != ToolSOPS {
		t.Fatalf("DetectTool() tool = %q, want %q", tool, ToolSOPS)
	}
}

func TestDetectToolFallsBackToAGE(t *testing.T) {
	origLookPath := lookPath
	t.Cleanup(func() { lookPath = origLookPath })

	lookPath = func(file string) (string, error) {
		switch file {
		case "age":
			return "/tmp/age", nil
		default:
			return "", errors.New("not found")
		}
	}

	tool, err := DetectTool()
	if err != nil {
		t.Fatalf("DetectTool() error = %v", err)
	}
	if tool != ToolAGE {
		t.Fatalf("DetectTool() tool = %q, want %q", tool, ToolAGE)
	}
}

func TestDetectToolMissing(t *testing.T) {
	origLookPath := lookPath
	t.Cleanup(func() { lookPath = origLookPath })

	lookPath = func(file string) (string, error) {
		return "", errors.New("not found")
	}

	_, err := DetectTool()
	if err == nil {
		t.Fatal("DetectTool() expected error, got nil")
	}
}

func TestDecryptFileUsesSOPS(t *testing.T) {
	origLookPath := lookPath
	origRunTool := runTool
	t.Cleanup(func() {
		lookPath = origLookPath
		runTool = origRunTool
	})

	lookPath = func(file string) (string, error) {
		if file == "sops" {
			return "/tmp/sops", nil
		}
		return "", errors.New("not found")
	}

	var calledName string
	var calledArgs []string
	runTool = func(name string, args ...string) ([]byte, error) {
		calledName = name
		calledArgs = args
		return []byte("plain\n"), nil
	}

	plain, tool, err := DecryptFile("configs/secrets/api.enc.yaml")
	if err != nil {
		t.Fatalf("DecryptFile() error = %v", err)
	}
	if tool != ToolSOPS {
		t.Fatalf("DecryptFile() tool = %q, want %q", tool, ToolSOPS)
	}
	if string(plain) != "plain\n" {
		t.Fatalf("DecryptFile() output = %q, want %q", plain, "plain\n")
	}
	if calledName != "sops" {
		t.Fatalf("called tool = %q, want sops", calledName)
	}
	if len(calledArgs) != 2 || calledArgs[0] != "--decrypt" || calledArgs[1] != "configs/secrets/api.enc.yaml" {
		t.Fatalf("called args = %#v, want [--decrypt configs/secrets/api.enc.yaml]", calledArgs)
	}
}
