package cmd

import (
	"testing"

	"github.com/rickyroynardson/codex-switch/internal/paths"
	"github.com/rickyroynardson/codex-switch/internal/state"
	"github.com/stretchr/testify/assert"
)

func TestRunRemoveRemovesNonActiveAccount(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)

	layout := paths.NewLayout(dir)
	registry := state.NewRegistry()
	registry.Accounts = []state.Account{
		{
			Tag: "personal",
		},
		{
			Tag: "work",
		},
	}
	registry.ActiveTag = "personal"
	err := state.SaveRegistry(layout.RegistryPath, registry)
	assert.NoError(t, err)

	cmd, out := newTestCommandOutput()

	err = runRemove(cmd, []string{"work"})
	assert.NoError(t, err)
	assert.Equal(t, "removed account work\n", out.String())

	registry, err = state.LoadRegistry(layout.RegistryPath)
	assert.NoError(t, err)

	_, ok := registry.FindAccount("personal")
	assert.True(t, ok)

	_, ok = registry.FindAccount("work")
	assert.False(t, ok)

	assert.Equal(t, "personal", registry.ActiveTag)
	assert.Len(t, registry.Accounts, 1)

}

func TestRunRemoveReturnsErrorForActiveAccount(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)

	layout := paths.NewLayout(dir)
	registry := state.NewRegistry()
	registry.Accounts = []state.Account{
		{
			Tag: "personal",
		},
	}
	registry.ActiveTag = "personal"
	err := state.SaveRegistry(layout.RegistryPath, registry)
	assert.NoError(t, err)

	cmd, _ := newTestCommandOutput()

	err = runRemove(cmd, []string{"personal"})
	assert.Error(t, err)
	assert.Equal(t, "cannot remove active account", err.Error())

	registry, err = state.LoadRegistry(layout.RegistryPath)
	assert.NoError(t, err)
	assert.Equal(t, "personal", registry.ActiveTag)
	assert.Len(t, registry.Accounts, 1)

	_, ok := registry.FindAccount("personal")
	assert.True(t, ok)
}

func TestRunRemoveReturnsErrorForMissingAccount(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)

	layout := paths.NewLayout(dir)
	registry := state.NewRegistry()
	registry.Accounts = []state.Account{
		{
			Tag: "personal",
		},
	}
	registry.ActiveTag = "personal"
	err := state.SaveRegistry(layout.RegistryPath, registry)
	assert.NoError(t, err)

	cmd, _ := newTestCommandOutput()

	err = runRemove(cmd, []string{"work"})
	assert.Error(t, err)
	assert.Equal(t, "account with tag: work not found", err.Error())

	registry, err = state.LoadRegistry(layout.RegistryPath)
	assert.NoError(t, err)
	assert.Equal(t, "personal", registry.ActiveTag)
	assert.Len(t, registry.Accounts, 1)

	_, ok := registry.FindAccount("personal")
	assert.True(t, ok)
}

func TestRunRemoveReturnsErrorForInvalidTag(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)

	layout := paths.NewLayout(dir)
	registry := state.NewRegistry()
	registry.Accounts = []state.Account{
		{
			Tag: "personal",
		},
	}
	registry.ActiveTag = "personal"
	err := state.SaveRegistry(layout.RegistryPath, registry)
	assert.NoError(t, err)

	cmd, _ := newTestCommandOutput()

	err = runRemove(cmd, []string{"invalid tag!"})
	assert.Error(t, err)

	registry, err = state.LoadRegistry(layout.RegistryPath)
	assert.NoError(t, err)
	assert.Equal(t, "personal", registry.ActiveTag)
	assert.Len(t, registry.Accounts, 1)

	_, ok := registry.FindAccount("personal")
	assert.True(t, ok)
}
