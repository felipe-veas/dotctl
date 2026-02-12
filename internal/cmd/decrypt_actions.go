package cmd

import (
	"github.com/felipe-veas/dotctl/internal/decrypt"
	"github.com/felipe-veas/dotctl/internal/manifest"
)

func countDecryptActions(actions []manifest.Action) int {
	count := 0
	for _, action := range actions {
		if action.Decrypt {
			count++
		}
	}
	return count
}

func detectDecryptToolForActions(actions []manifest.Action) (decrypt.Tool, int, error) {
	count := countDecryptActions(actions)
	if count == 0 {
		return "", 0, nil
	}

	tool, err := decrypt.DetectTool()
	if err != nil {
		return "", count, err
	}
	return tool, count, nil
}
