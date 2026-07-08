package runtimehome

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/rickyroynardson/codex-switch/internal/codex"
	"github.com/rickyroynardson/codex-switch/internal/paths"
	"github.com/rickyroynardson/codex-switch/internal/state"
)

var SharedEntryNames = []string{
	"config.toml",
	"session_index.jsonl",
	"history.jsonl",
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

var requiredSharedFiles = map[string]string{
	"config.toml":         codex.FileAuthConfig + "\n",
	"session_index.jsonl": "",
	"history.jsonl":       "",
}

var requiredSharedDirs = map[string]struct{}{
	"sessions":          {},
	"archived_sessions": {},
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

		if _, ok := requiredSharedDirs[name]; ok {
			if err := os.MkdirAll(sharedPath, 0700); err != nil {
				return err
			}
		}

		if contents, ok := requiredSharedFiles[name]; ok {
			if err := ensureSharedFile(sharedPath, contents); err != nil {
				return err
			}
		}

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

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		target := filepath.Join(dst, rel)

		info, err := entry.Info()
		if err != nil {
			return err
		}

		if entry.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}

		return copyFile(path, target, info.Mode().Perm())
	})
}

func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return copyDir(src, dst)
	}

	return copyFile(src, dst, info.Mode().Perm())
}

func ensureSharedFile(path, contents string) error {
	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			return fmt.Errorf("shared file %s is a directory", path)
		}

		if filepath.Base(path) == "config.toml" {
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			normalized := normalizeSharedConfigContents(string(b))
			return os.WriteFile(path, []byte(normalized+"\n"), 0600)
		}

		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	if filepath.Base(path) == "config.toml" {
		contents = normalizeSharedConfigContents(contents) + "\n"
	}

	return os.WriteFile(path, []byte(contents), 0600)
}

func ImportSharedState(layout paths.Layout, sourceHome string) error {
	if _, err := os.Stat(sourceHome); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if err := os.MkdirAll(layout.SharedDir, 0700); err != nil {
		return err
	}

	for _, name := range SharedEntryNames {
		sourcePath := filepath.Join(sourceHome, name)
		targetPath := filepath.Join(layout.SharedDir, name)

		if _, err := os.Stat(sourcePath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}

		if name == "config.toml" {
			if err := copyNormalizedConfig(sourcePath, targetPath); err != nil {
				return err
			}
			continue
		}

		if err := copyPath(sourcePath, targetPath); err != nil {
			return err
		}
	}

	return nil
}

func PersistSharedState(layout paths.Layout) error {
	entries, err := os.ReadDir(layout.CurrentHomeDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if err := os.MkdirAll(layout.SharedDir, 0700); err != nil {
		return err
	}

	for _, entry := range entries {
		name := entry.Name()
		if !IsSharedEntry(name) {
			continue
		}

		runtimePath := filepath.Join(layout.CurrentHomeDir, name)

		info, err := os.Lstat(runtimePath)
		if err != nil {
			return err
		}

		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}

		sharedPath := filepath.Join(layout.SharedDir, name)

		if name == "config.toml" {
			if err := copyNormalizedConfig(runtimePath, sharedPath); err != nil {
				return err
			}
			continue
		}

		if err := copyPath(runtimePath, sharedPath); err != nil {
			return err
		}
	}

	return nil
}

func normalizeSharedConfigContents(contents string) string {
	lines := strings.Split(contents, "\n")
	sanitized := make([]string, 0, len(lines))

	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "cli_auth_credentials_store = ") {
			continue
		}
		sanitized = append(sanitized, line)
	}

	firstTableIndex := -1
	for i, line := range sanitized {
		if strings.HasPrefix(strings.TrimSpace(line), "[") {
			firstTableIndex = i
			break
		}
	}

	authLine := strings.TrimSpace(codex.FileAuthConfig)

	if firstTableIndex == -1 {
		topLevel := strings.TrimSpace(strings.Join(sanitized, "\n"))
		if topLevel == "" {
			return authLine
		}
		return topLevel + "\n" + authLine
	}

	topLevel := strings.TrimSpace(strings.Join(sanitized[:firstTableIndex], "\n"))
	tables := strings.TrimSpace(strings.Join(sanitized[firstTableIndex:], "\n"))

	if topLevel == "" {
		return authLine + "\n\n" + tables
	}

	return topLevel + "\n" + authLine + "\n\n" + tables
}

func copyNormalizedConfig(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
		return err
	}

	normalized := normalizeSharedConfigContents(string(b))
	return os.WriteFile(dst, []byte(normalized+"\n"), 0600)
}
