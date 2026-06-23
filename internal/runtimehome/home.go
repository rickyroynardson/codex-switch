package runtimehome

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/rickyroynardson/codex-switch/internal/paths"
	"github.com/rickyroynardson/codex-switch/internal/state"
)

var SharedEntryNames = []string{
	"config.toml",
	"sessions",
	"archived_sessions",
	"skills",
	"themes",
	"AGENTS.md",
	"AGENTS.override.md",
	"rules",
	"hooks",
	"prompts",
	"plugins",
}

func IsSharedEntry(name string) bool {
	return slices.Contains(SharedEntryNames, name)
}

func Assemble(layout paths.Layout, account state.Account) error {
	if account.AuthPath == "" {
		return fmt.Errorf("account %s has empty auth path", account.Tag)
	}

	if _, err := os.Stat(account.AuthPath); err != nil {
		return fmt.Errorf("account auth file not found: %w", err)
	}

	if err := os.MkdirAll(layout.RuntimeDir, 0700); err != nil {
		return err
	}

	stagingDir := filepath.Join(layout.RuntimeDir, fmt.Sprintf("current-home.%d.%d", os.Getpid(), time.Now().UnixNano()))
	if err := os.MkdirAll(stagingDir, 0700); err != nil {
		return err
	}

	cleanup := true
	defer func() {
		if cleanup {
			_ = os.RemoveAll(stagingDir)
		}
	}()

	for _, name := range SharedEntryNames {
		sharedPath := filepath.Join(layout.SharedDir, name)
		if _, err := os.Stat(sharedPath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}

		runtimePath := filepath.Join(stagingDir, name)
		if err := os.Symlink(sharedPath, runtimePath); err != nil {
			return err
		}
	}

	authPath := filepath.Join(stagingDir, "auth.json")
	if err := copyFile(account.AuthPath, authPath, 0600); err != nil {
		return err
	}

	if err := os.RemoveAll(layout.CurrentHomeDir); err != nil {
		return err
	}

	if err := os.Rename(stagingDir, layout.CurrentHomeDir); err != nil {
		return err
	}

	cleanup = false
	return nil
}

func copyFile(src, dst string, perm os.FileMode) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, perm)
}
