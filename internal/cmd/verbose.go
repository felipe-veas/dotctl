package cmd

import (
	"fmt"
	"os"
)

func verbosef(format string, args ...any) {
	if !flagVerbose {
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, "[verbose] "+format+"\n", args...)
}
