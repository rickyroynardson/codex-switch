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

func stubEnsureAccountHome(t *testing.T, fn func(paths.Layout, string) error) {
	t.Helper()

	old := ensureAccountHome
	t.Cleanup(func() { ensureAccountHome = old })
	ensureAccountHome = fn
}

func stubRunCodexWithHome(t *testing.T, fn func(codex.RunOptions) error) {
	t.Helper()

	old := runCodexWithHome
	t.Cleanup(func() { runCodexWithHome = old })
	runCodexWithHome = fn
}

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
	require.NoError(t, state.SaveRegistry(layout.RegistryPath, registry))

	ensured := false
	stubEnsureAccountHome(t, func(gotLayout paths.Layout, tag string) error {
		ensured = true
		assert.Equal(t, "work", tag)
		return nil
	})

	ranCodex := false
	stubRunCodexWithHome(t, func(opts codex.RunOptions) error {
		ranCodex = true
		assert.Equal(t, layout.AccountDir("work"), opts.CodexHome)
		assert.Equal(t, []string{"status"}, opts.Args)
		assert.Equal(t, "/real/codex", opts.CodexCommand)
		return nil
	})

	cmd, _ := newTestCommandOutput()

	require.NoError(t, runProxy(cmd, []string{"status"}))
	assert.True(t, ensured)
	assert.True(t, ranCodex)
}

func TestRunProxyReturnsErrorWhenNoActiveAccount(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)

	layout := paths.NewLayout(dir)
	require.NoError(t, state.SaveRegistry(layout.RegistryPath, state.NewRegistry()))

	stubEnsureAccountHome(t, func(paths.Layout, string) error {
		t.Fatalf("ensureAccountHome should not be called")
		return nil
	})
	stubRunCodexWithHome(t, func(codex.RunOptions) error {
		t.Fatalf("runCodexWithHome should not be called")
		return nil
	})

	cmd, _ := newTestCommandOutput()

	err := runProxy(cmd, []string{"status"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no active account")
}

func TestRunProxyReturnsErrorWhenEnsureAccountHomeFails(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)

	layout := paths.NewLayout(dir)
	registry := state.NewRegistry()
	registry.UpsertAccount(state.Account{
		Tag:      "work",
		AuthPath: layout.AccountAuthPath("work"),
	})
	require.NoError(t, state.SaveRegistry(layout.RegistryPath, registry))

	stubEnsureAccountHome(t, func(paths.Layout, string) error {
		return errors.New("ensure error")
	})
	stubRunCodexWithHome(t, func(codex.RunOptions) error {
		t.Fatalf("runCodexWithHome should not be called")
		return nil
	})

	cmd, _ := newTestCommandOutput()

	err := runProxy(cmd, []string{"status"})
	require.Error(t, err)
	assert.Equal(t, "ensure error", err.Error())
}

func TestRunProxyReturnsErrorWhenCodexRunFails(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)
	t.Setenv(EnvRealCodex, "/real/codex")

	layout := paths.NewLayout(dir)
	registry := state.NewRegistry()
	registry.UpsertAccount(state.Account{
		Tag:      "work",
		AuthPath: layout.AccountAuthPath("work"),
	})
	require.NoError(t, state.SaveRegistry(layout.RegistryPath, registry))

	stubEnsureAccountHome(t, func(paths.Layout, string) error { return nil })
	stubRunCodexWithHome(t, func(codex.RunOptions) error {
		return errors.New("run error")
	})

	cmd, _ := newTestCommandOutput()

	err := runProxy(cmd, []string{"status"})
	require.Error(t, err)
	assert.Equal(t, "run error", err.Error())
}
