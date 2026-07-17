package accounthome

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rickyroynardson/codex-switch/internal/codex"
	"github.com/rickyroynardson/codex-switch/internal/paths"
)

// SharedEntryNames are the Codex home entries shared across every account.
// Each is symlinked from the account dir into shared/, so a user's session
// history follows them when they switch accounts. All entries are either
// directories or append-only files, which Codex writes through a symlink
// without clobbering it — so no copy-back step is needed.
var SharedEntryNames = []string{
	"sessions",
	"archived_sessions",
	"history.jsonl",
	"session_index.jsonl",
}

var sharedDirs = map[string]bool{
	"sessions":          true,
	"archived_sessions": true,
}

// EnsureAccountHome makes accounts/<tag> a usable CODEX_HOME: it guarantees the
// file-auth config line (per-account, never truncated) and symlinks the shared
// session entries into the account dir. Idempotent — proxy and login both call
// it. It never clobbers a real file sitting where a shared symlink belongs.
func EnsureAccountHome(layout paths.Layout, tag string) error {
	accountDir := layout.AccountDir(tag)
	if err := os.MkdirAll(accountDir, 0700); err != nil {
		return err
	}

	if err := codex.EnsureFileAuthConfig(accountDir); err != nil {
		return err
	}

	if err := os.MkdirAll(layout.SharedDir, 0700); err != nil {
		return err
	}

	for _, name := range SharedEntryNames {
		sharedPath := filepath.Join(layout.SharedDir, name)

		if sharedDirs[name] {
			if err := os.MkdirAll(sharedPath, 0700); err != nil {
				return err
			}
		} else if err := ensureEmptyFile(sharedPath); err != nil {
			return err
		}

		if err := ensureSymlink(filepath.Join(accountDir, name), sharedPath); err != nil {
			return err
		}
	}

	return nil
}

// ImportSharedState seeds shared/ from an existing Codex home (e.g. ~/.codex),
// copying only the shared session entries. Sensitive files (auth.json, tokens)
// and per-account config are never copied.
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
		if _, err := os.Stat(sourcePath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}

		if err := copyPath(sourcePath, filepath.Join(layout.SharedDir, name)); err != nil {
			return err
		}
	}

	return nil
}

func ensureEmptyFile(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	return os.WriteFile(path, nil, 0600)
}

func ensureSymlink(linkPath, target string) error {
	info, err := os.Lstat(linkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return os.Symlink(target, linkPath)
		}
		return err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		current, err := os.Readlink(linkPath)
		if err != nil {
			return err
		}
		if current == target {
			return nil
		}
		if err := os.Remove(linkPath); err != nil {
			return err
		}
		return os.Symlink(target, linkPath)
	}

	// ponytail: a real file/dir sits where a shared symlink must go (only
	// reachable for an account created before this layout existed). Refuse
	// rather than clobber; add merge-into-shared here if anyone hits it.
	return fmt.Errorf("cannot share %q: real file exists at %s; move or remove it", filepath.Base(linkPath), linkPath)
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

func copyFile(src, dst string, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
		return err
	}

	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, b, perm)
}
