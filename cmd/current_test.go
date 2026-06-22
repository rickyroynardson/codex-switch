package cmd

import (
	"bytes"
	"testing"

	"github.com/rickyroynardson/codex-switch/internal/paths"
	"github.com/rickyroynardson/codex-switch/internal/state"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestRunCurrentPrintsNoneWhenNoActiveAccount(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)

	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)

	err := runCurrent(cmd, nil)
	assert.NoError(t, err)
	assert.Equal(t, "none\n", out.String())
}

func TestRunCurrentPrintsActiveAccount(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)

	layout := paths.NewLayout(dir)
	registry := state.NewRegistry()
	acc := state.Account{
		Tag:       "work",
		AuthPath:  "/tmp/new-auth.json",
		Email:     "work@mail.com",
		AuthState: state.AuthStateReady,
		CreatedAt: "2026-06-12T00:00:00Z",
		UpdatedAt: "2026-06-12T00:00:00Z",
	}
	registry.UpsertAccount(acc)
	err := state.SaveRegistry(layout.RegistryPath, registry)
	assert.NoError(t, err)

	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)

	err = runCurrent(cmd, nil)
	assert.NoError(t, err)
	assert.Equal(t, "work\n", out.String())
}
