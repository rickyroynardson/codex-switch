package cmd

import (
	"os"

	"github.com/rickyroynardson/codex-switch/internal/paths"
)

func realCodexCommand(layout paths.Layout) (string, error) {
	if real := os.Getenv(EnvRealCodex); real != "" {
		return real, nil
	}
	return findRealCodex(layout.WrapperPath, os.Getenv("PATH"))
}
