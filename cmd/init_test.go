package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rickyroynardson/codex-switch/internal/paths"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunInitInstallsWrapper(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)
	t.Setenv("PATH", "/fake/bin")

	layout := paths.NewLayout(dir)

	oldFindRealCodex := findRealCodex
	t.Cleanup(func() {
		findRealCodex = oldFindRealCodex
	})
	findRealCodex = func(wrapperPath, pathEnv string) (string, error) {
		assert.Equal(t, layout.WrapperPath, wrapperPath)
		assert.NotEmpty(t, pathEnv)
		assert.Equal(t, "/fake/bin", pathEnv)
		return "/real/codex", nil
	}

	oldInstallWrapper := installWrapper
	t.Cleanup(func() {
		installWrapper = oldInstallWrapper
	})
	installedWrapperPath := ""
	installedRealCodexPath := ""
	installedLauncher := ""
	installWrapper = func(wrapperPath, realCodexPath, launcher string) error {
		installedWrapperPath = wrapperPath
		installedRealCodexPath = realCodexPath
		installedLauncher = launcher
		return nil
	}

	oldUserHomeDir := userHomeDir
	t.Cleanup(func() {
		userHomeDir = oldUserHomeDir
	})
	userHomeDir = func() (string, error) {
		return t.TempDir(), nil
	}

	oldImportSharedState := importSharedState
	t.Cleanup(func() {
		importSharedState = oldImportSharedState
	})
	importSharedState = func(layout paths.Layout, sourceHome string) error {
		t.Fatalf("imporSharedState should not be called")
		return nil
	}

	cmd, out := newTestCommandOutput()

	err := runInit(cmd, nil)
	require.NoError(t, err)

	assert.Equal(t, layout.WrapperPath, installedWrapperPath)
	assert.Equal(t, "/real/codex", installedRealCodexPath)
	assert.Equal(t, "codex-switch", installedLauncher)

	output := out.String()
	assert.Contains(t, output, "installed wrapper")
	assert.Contains(t, output, layout.WrapperPath)
	assert.Contains(t, output, "real codex")
	assert.Contains(t, output, "/real/codex")
	assert.Contains(t, output, "add to PATH")
	assert.Contains(t, output, layout.BinDir)
}

func TestRunInitReturnsErrorWhenRealCodexNotFound(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)

	oldFindRealCodex := findRealCodex
	t.Cleanup(func() {
		findRealCodex = oldFindRealCodex
	})
	findRealCodex = func(wrapperPath, pathEnv string) (string, error) {
		return "", assert.AnError
	}

	oldInstallWrapper := installWrapper
	t.Cleanup(func() {
		installWrapper = oldInstallWrapper
	})
	installWrapper = func(wrapperPath, realCodexPath, launcher string) error {
		t.Fatalf("installWrapper should not be called")
		return nil
	}

	cmd, _ := newTestCommandOutput()

	err := runInit(cmd, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError)
}

func TestRunInitReturnsErrorWhenInstallFails(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)

	oldFindRealCodex := findRealCodex
	t.Cleanup(func() {
		findRealCodex = oldFindRealCodex
	})
	findRealCodex = func(wrapperPath, pathEnv string) (string, error) {
		return "/real/codex", nil
	}

	oldInstallWrapper := installWrapper
	t.Cleanup(func() {
		installWrapper = oldInstallWrapper
	})
	installWrapper = func(wrapperPath, realCodexPath, launcher string) error {
		return assert.AnError
	}

	cmd, _ := newTestCommandOutput()

	err := runInit(cmd, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError)
}
func TestRunInitImportsSharedStateWhenSharedIsEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)
	t.Setenv("PATH", "/fake/bin")

	layout := paths.NewLayout(dir)

	userHome := t.TempDir()
	sourceHome := filepath.Join(userHome, ".codex")
	require.NoError(t, os.MkdirAll(sourceHome, 0700))

	oldUserHomeDir := userHomeDir
	t.Cleanup(func() {
		userHomeDir = oldUserHomeDir
	})
	userHomeDir = func() (string, error) {
		return userHome, nil
	}

	oldFindRealCodex := findRealCodex
	t.Cleanup(func() {
		findRealCodex = oldFindRealCodex
	})
	findRealCodex = func(wrapperPath, pathEnv string) (string, error) {
		return "/real/codex", nil
	}

	oldInstallWrapper := installWrapper
	t.Cleanup(func() {
		installWrapper = oldInstallWrapper
	})
	installWrapper = func(wrapperPath, realCodexPath, launcher string) error {
		return nil
	}

	oldImportSharedState := importSharedState
	t.Cleanup(func() {
		importSharedState = oldImportSharedState
	})

	imported := false
	importSharedState = func(gotLayout paths.Layout, gotSourceHome string) error {
		imported = true
		assert.Equal(t, layout.SharedDir, gotLayout.SharedDir)
		assert.Equal(t, sourceHome, gotSourceHome)
		return nil
	}

	cmd, out := newTestCommandOutput()

	err := runInit(cmd, nil)
	require.NoError(t, err)

	assert.True(t, imported)
	assert.Contains(t, out.String(), "imported shared state")
	assert.Contains(t, out.String(), sourceHome)
}

func TestRunInitSkipsSharedImportWhenSharedIsNotEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)

	layout := paths.NewLayout(dir)
	require.NoError(t, os.MkdirAll(layout.SharedDir, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(layout.SharedDir, "config.toml"), []byte("existing"), 0600))

	oldFindRealCodex := findRealCodex
	t.Cleanup(func() {
		findRealCodex = oldFindRealCodex
	})
	findRealCodex = func(wrapperPath, pathEnv string) (string, error) {
		return "/real/codex", nil
	}

	oldInstallWrapper := installWrapper
	t.Cleanup(func() {
		installWrapper = oldInstallWrapper
	})
	installWrapper = func(wrapperPath, realCodexPath, launcher string) error {
		return nil
	}

	oldImportSharedState := importSharedState
	t.Cleanup(func() {
		importSharedState = oldImportSharedState
	})
	importSharedState = func(gotLayout paths.Layout, gotSourceHome string) error {
		t.Fatalf("importSharedState should not be called")
		return nil
	}

	cmd, _ := newTestCommandOutput()

	err := runInit(cmd, nil)
	require.NoError(t, err)
}
