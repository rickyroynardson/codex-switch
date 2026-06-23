package cmd

import (
	"testing"

	"github.com/rickyroynardson/codex-switch/internal/paths"
	"github.com/rickyroynardson/codex-switch/internal/state"
	"github.com/stretchr/testify/assert"
)

func TestRunSwitchSetsActiveAccount(t *testing.T) {
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
	err := state.SaveRegistry(layout.RegistryPath, registry)
	assert.NoError(t, err)

	cmd, out := newTestCommandOutput()

	err = runSwitch(cmd, []string{"work"})
	assert.NoError(t, err)

	registry, err = state.LoadRegistry(layout.RegistryPath)
	assert.NoError(t, err)
	assert.Equal(t, "work", registry.ActiveTag)
	assert.Equal(t, "switched to work\n", out.String())
}

func TestRunSwitchReturnsErrorForMissingAccount(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)

	layout := paths.NewLayout(dir)
	registry := state.NewRegistry()
	registry.UpsertAccount(state.Account{
		Tag: "personal",
	})
	err := state.SaveRegistry(layout.RegistryPath, registry)
	assert.NoError(t, err)

	cmd, _ := newTestCommandOutput()

	err = runSwitch(cmd, []string{"work"})
	assert.Error(t, err)

	registry, err = state.LoadRegistry(layout.RegistryPath)
	assert.NoError(t, err)
	assert.Equal(t, "personal", registry.ActiveTag)
}

func TestRunSwitchReturnsErrorForInvalidTag(t *testing.T) {
	cmd, _ := newTestCommandOutput()
	err := runSwitch(cmd, []string{"../bad"})
	assert.Error(t, err)
}
