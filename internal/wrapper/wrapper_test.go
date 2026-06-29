package wrapper

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindRealCodexFindsExecutable(t *testing.T) {
	dir := t.TempDir()
	realCodexPath := filepath.Join(dir, "codex")

	writeExecutable(t, realCodexPath)

	got, err := FindRealCodex(filepath.Join(t.TempDir(), "codex"), dir)
	require.NoError(t, err)
	assert.Equal(t, realCodexPath, got)
}

func TestFindRealCodexSkipsWrapperPath(t *testing.T) {
	wrapperDir := t.TempDir()
	wrapperPath := filepath.Join(wrapperDir, "codex")

	realDir := t.TempDir()
	realCodexPath := filepath.Join(realDir, "codex")

	writeExecutable(t, wrapperPath)
	writeExecutable(t, realCodexPath)

	pathEnv := wrapperDir + string(os.PathListSeparator) + realDir

	got, err := FindRealCodex(wrapperPath, pathEnv)
	require.NoError(t, err)
	assert.Equal(t, realCodexPath, got)
}

func TestFindRealCodexReturnsErrorWhenMissing(t *testing.T) {
	dir := t.TempDir()

	got, err := FindRealCodex(filepath.Join(dir, "codex"), dir)
	require.Error(t, err)
	assert.Empty(t, got)
	assert.Contains(t, err.Error(), "real codex executable not found")
}

func TestFindRealCodexIgnoresNonExecutable(t *testing.T) {
	dir := t.TempDir()
	codexPath := filepath.Join(dir, "codex")

	if err := os.WriteFile(codexPath, []byte("#!/bin/sh\n"), 0600); err != nil {
		t.Fatalf("write non-executable codex: %v", err)
	}

	got, err := FindRealCodex(filepath.Join(t.TempDir(), "codex"), dir)
	require.Error(t, err)
	assert.Empty(t, got)
	assert.Contains(t, err.Error(), "real codex executable not found")
}

func writeExecutable(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0700); err != nil {
		t.Fatalf("write executable: %v", err)
	}
}

func TestInstallWritesExecutableWrapper(t *testing.T) {
	dir := t.TempDir()

	wrapperPath := filepath.Join(dir, "bin", "codex")
	realCodexPath := filepath.Join(dir, "real", "codex")

	err := Install(wrapperPath, realCodexPath, "codex-switch")
	require.NoError(t, err)

	info, err := os.Stat(wrapperPath)
	require.NoError(t, err)

	assert.False(t, info.IsDir())
	assert.NotZero(t, info.Mode()&0111)

	b, err := os.ReadFile(wrapperPath)
	require.NoError(t, err)

	contents := string(b)
	assert.Contains(t, contents, "#!/bin/sh\n")
	assert.Contains(t, contents, "\nexport CODEX_SWITCH_REAL_CODEX=")
	assert.Contains(t, contents, realCodexPath)
	assert.Contains(t, contents, "\nexec 'codex-switch' proxy \"$@\"")
}

func TestInstallRequiresWrapperPath(t *testing.T) {
	err := Install("", "/real/codex", "codex-switch")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "wrapper path is required")
}

func TestInstallRequiresRealCodexPath(t *testing.T) {
	dir := t.TempDir()
	wrapperPath := filepath.Join(dir, "bin", "codex")
	err := Install(wrapperPath, "", "codex-switch")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "real codex path is required")
}

func TestInstallUsesDefaultLauncher(t *testing.T) {
	dir := t.TempDir()
	wrapperPath := filepath.Join(dir, "bin", "codex")
	realCodexPath := filepath.Join(dir, "real", "codex")

	err := Install(wrapperPath, realCodexPath, "")
	require.NoError(t, err)

	b, err := os.ReadFile(wrapperPath)
	require.NoError(t, err)

	assert.Contains(t, string(b), "exec 'codex-switch' proxy \"$@\"")
}

func TestShellQuoteHandlesSingleQuote(t *testing.T) {
	got := shellQuote("/some path/it's/codex")
	assert.Equal(t, `'/some path/it'"'"'s/codex'`, got)
}
