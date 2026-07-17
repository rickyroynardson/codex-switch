package accounthome

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rickyroynardson/codex-switch/internal/codex"
	"github.com/rickyroynardson/codex-switch/internal/paths"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureAccountHomeWritesFileAuthConfig(t *testing.T) {
	layout := paths.NewLayout(t.TempDir())

	require.NoError(t, EnsureAccountHome(layout, "work"))

	b, err := os.ReadFile(filepath.Join(layout.AccountDir("work"), "config.toml"))
	require.NoError(t, err)
	assert.Contains(t, string(b), codex.FileAuthConfig)
}

func TestEnsureAccountHomeLinksSharedEntries(t *testing.T) {
	layout := paths.NewLayout(t.TempDir())

	require.NoError(t, EnsureAccountHome(layout, "work"))

	for _, name := range SharedEntryNames {
		link := filepath.Join(layout.AccountDir("work"), name)

		info, err := os.Lstat(link)
		require.NoError(t, err, name)
		assert.NotZero(t, info.Mode()&os.ModeSymlink, name)

		target, err := os.Readlink(link)
		require.NoError(t, err, name)
		assert.Equal(t, filepath.Join(layout.SharedDir, name), target, name)
	}
}

func TestEnsureAccountHomeCreatesSharedTargets(t *testing.T) {
	layout := paths.NewLayout(t.TempDir())

	require.NoError(t, EnsureAccountHome(layout, "work"))

	for name, isDir := range sharedDirs {
		info, err := os.Stat(filepath.Join(layout.SharedDir, name))
		require.NoError(t, err, name)
		assert.Equal(t, isDir, info.IsDir(), name)
	}

	// history is written to across accounts, so it must exist to be symlinked.
	_, err := os.Stat(filepath.Join(layout.SharedDir, "history.jsonl"))
	require.NoError(t, err)
}

func TestEnsureAccountHomeSharesHistoryAcrossAccounts(t *testing.T) {
	layout := paths.NewLayout(t.TempDir())

	require.NoError(t, EnsureAccountHome(layout, "work"))
	require.NoError(t, EnsureAccountHome(layout, "personal"))

	// A write through one account's symlink lands in shared/, visible to the other.
	require.NoError(t, os.WriteFile(filepath.Join(layout.AccountDir("work"), "history.jsonl"), []byte("shared line\n"), 0600))

	b, err := os.ReadFile(filepath.Join(layout.AccountDir("personal"), "history.jsonl"))
	require.NoError(t, err)
	assert.Equal(t, "shared line\n", string(b))
}

func TestEnsureAccountHomeIsIdempotent(t *testing.T) {
	layout := paths.NewLayout(t.TempDir())

	require.NoError(t, EnsureAccountHome(layout, "work"))
	require.NoError(t, EnsureAccountHome(layout, "work"))

	link := filepath.Join(layout.AccountDir("work"), "sessions")
	target, err := os.Readlink(link)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(layout.SharedDir, "sessions"), target)
}

func TestEnsureAccountHomeRefusesToClobberRealEntry(t *testing.T) {
	layout := paths.NewLayout(t.TempDir())

	accountDir := layout.AccountDir("work")
	require.NoError(t, os.MkdirAll(accountDir, 0700))
	// A real sessions dir where a shared symlink should go.
	require.NoError(t, os.MkdirAll(filepath.Join(accountDir, "sessions"), 0700))

	err := EnsureAccountHome(layout, "work")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot share")
}

func TestImportSharedStateCopiesSessionEntries(t *testing.T) {
	layout := paths.NewLayout(t.TempDir())
	sourceHome := filepath.Join(t.TempDir(), ".codex")

	require.NoError(t, os.MkdirAll(filepath.Join(sourceHome, "sessions"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(sourceHome, "sessions", "one.jsonl"), []byte("{}\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(sourceHome, "history.jsonl"), []byte("h\n"), 0600))

	require.NoError(t, ImportSharedState(layout, sourceHome))

	session, err := os.ReadFile(filepath.Join(layout.SharedDir, "sessions", "one.jsonl"))
	require.NoError(t, err)
	assert.Equal(t, "{}\n", string(session))

	history, err := os.ReadFile(filepath.Join(layout.SharedDir, "history.jsonl"))
	require.NoError(t, err)
	assert.Equal(t, "h\n", string(history))
}

func TestImportSharedStateSkipsSensitiveAndPerAccountEntries(t *testing.T) {
	layout := paths.NewLayout(t.TempDir())
	sourceHome := filepath.Join(t.TempDir(), ".codex")

	require.NoError(t, os.MkdirAll(sourceHome, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(sourceHome, "auth.json"), []byte("secret"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(sourceHome, "config.toml"), []byte("model=1"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(sourceHome, "logs_2.sqlite"), []byte("logs"), 0600))

	require.NoError(t, ImportSharedState(layout, sourceHome))

	for _, name := range []string{"auth.json", "config.toml", "logs_2.sqlite"} {
		_, err := os.Stat(filepath.Join(layout.SharedDir, name))
		assert.True(t, os.IsNotExist(err), name)
	}
}

func TestImportSharedStateIgnoresMissingSource(t *testing.T) {
	layout := paths.NewLayout(t.TempDir())
	require.NoError(t, ImportSharedState(layout, filepath.Join(t.TempDir(), ".codex-missing")))
}
