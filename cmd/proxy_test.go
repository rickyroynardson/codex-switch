package cmd

import (
	"errors"
	"testing"

	"github.com/rickyroynardson/codex-switch/internal/codex"
	"github.com/rickyroynardson/codex-switch/internal/paths"
	"github.com/rickyroynardson/codex-switch/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunProxyRunsCodexWithActiveAccount(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)
	t.Setenv(EnvRealCodex, "/real/codex")

	layout := paths.NewLayout(dir)
	registry := state.NewRegistry()
	registry.UpsertAccount(state.Account{
		Tag:      "work",
		AuthPath: layout.AccountAuthPath("work"),
	})

	err := state.SaveRegistry(layout.RegistryPath, registry)
	require.NoError(t, err)

	assembled := false
	ranCodex := false

	oldAssembleRuntimeHome := assembleRuntimeHome
	t.Cleanup(func() {
		assembleRuntimeHome = oldAssembleRuntimeHome
	})

	assembleRuntimeHome = func(layout paths.Layout, account state.Account) error {
		assembled = true
		assert.Equal(t, "work", account.Tag)
		return nil
	}

	oldRunCodexWithHome := runCodexWithHome
	t.Cleanup(func() {
		runCodexWithHome = oldRunCodexWithHome
	})

	runCodexWithHome = func(opts codex.RunOptions) error {
		ranCodex = true
		assert.Equal(t, layout.CurrentHomeDir, opts.CodexHome)
		assert.Equal(t, []string{"status"}, opts.Args)
		assert.Equal(t, "/real/codex", opts.CodexCommand)
		return nil
	}

	cmd, _ := newTestCommandOutput()

	err = runProxy(cmd, []string{"status"})
	require.NoError(t, err)
	assert.True(t, assembled)
	assert.True(t, ranCodex)
}

func TestRunProxyReturnsErrorWhenNoActiveAccount(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)

	layout := paths.NewLayout(dir)
	registry := state.NewRegistry()
	err := state.SaveRegistry(layout.RegistryPath, registry)
	require.NoError(t, err)

	oldAssembleRuntimeHome := assembleRuntimeHome
	t.Cleanup(func() {
		assembleRuntimeHome = oldAssembleRuntimeHome
	})

	assembleRuntimeHome = func(layout paths.Layout, account state.Account) error {
		t.Fatalf("assembleRuntimeHome should not be called")
		return nil
	}

	oldRunCodexWithHome := runCodexWithHome
	t.Cleanup(func() {
		runCodexWithHome = oldRunCodexWithHome
	})

	runCodexWithHome = func(opts codex.RunOptions) error {
		t.Fatalf("runCodexWithHome should not be called")
		return nil
	}

	cmd, _ := newTestCommandOutput()

	err = runProxy(cmd, []string{"status"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no active account")
}

func TestRunProxyReturnsErrorWhenAssembleFails(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)

	layout := paths.NewLayout(dir)
	registry := state.NewRegistry()
	registry.UpsertAccount(state.Account{
		Tag:      "work",
		AuthPath: layout.AccountAuthPath("work"),
	})
	err := state.SaveRegistry(layout.RegistryPath, registry)
	require.NoError(t, err)

	oldAssembleRuntimeHome := assembleRuntimeHome
	t.Cleanup(func() {
		assembleRuntimeHome = oldAssembleRuntimeHome
	})
	assembleRuntimeHome = func(layout paths.Layout, account state.Account) error {
		return errors.New("assemble error")
	}

	oldRunCodexWithHome := runCodexWithHome
	t.Cleanup(func() {
		runCodexWithHome = oldRunCodexWithHome
	})
	runCodexWithHome = func(opts codex.RunOptions) error {
		t.Fatalf("runCodexWithHome should not be called")
		return nil
	}

	cmd, _ := newTestCommandOutput()

	err = runProxy(cmd, []string{"status"})
	require.Error(t, err)
	assert.Equal(t, "assemble error", err.Error())
}

func TestRunProxyReturnsErrorWhenCodexRunFails(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)

	layout := paths.NewLayout(dir)
	registry := state.NewRegistry()
	registry.UpsertAccount(state.Account{
		Tag:      "work",
		AuthPath: layout.AccountAuthPath("work"),
	})
	err := state.SaveRegistry(layout.RegistryPath, registry)
	require.NoError(t, err)

	oldAssembleRuntimeHome := assembleRuntimeHome
	t.Cleanup(func() {
		assembleRuntimeHome = oldAssembleRuntimeHome
	})
	assembleRuntimeHome = func(layout paths.Layout, account state.Account) error {
		return nil
	}

	oldRunCodexWithHome := runCodexWithHome
	t.Cleanup(func() {
		runCodexWithHome = oldRunCodexWithHome
	})
	runCodexWithHome = func(opts codex.RunOptions) error {
		return errors.New("run error")
	}

	cmd, _ := newTestCommandOutput()

	err = runProxy(cmd, []string{"status"})
	require.Error(t, err)
	assert.Equal(t, "run error", err.Error())
}
