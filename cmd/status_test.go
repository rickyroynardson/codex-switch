package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rickyroynardson/codex-switch/internal/paths"
	"github.com/rickyroynardson/codex-switch/internal/state"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func newTestCommandOutput() (*cobra.Command, *bytes.Buffer) {
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	return cmd, &out
}

func TestRunStatusPrintsNoAccounts(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)

	cmd, out := newTestCommandOutput()

	err := runStatus(cmd, nil)
	assert.NoError(t, err)
	assert.Equal(t, "no accounts\n", out.String())
}

func TestRunStatusPrintsAccounts(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)

	layout := paths.NewLayout(dir)
	registry := state.NewRegistry()
	acc := state.Account{
		Tag:       "test",
		AuthPath:  "/tmp/new-auth.json",
		Email:     "test@mail.com",
		AuthState: state.AuthStateReady,
		CreatedAt: "2026-06-12T00:00:00Z",
		UpdatedAt: "2026-06-12T00:00:00Z",
	}
	registry.UpsertAccount(acc)
	err := state.SaveRegistry(layout.RegistryPath, registry)
	assert.NoError(t, err)

	cmd, out := newTestCommandOutput()

	err = runStatus(cmd, nil)
	assert.NoError(t, err)
	assert.Contains(t, out.String(), "test")
	assert.Contains(t, out.String(), "test@mail.com")
	assert.Contains(t, out.String(), "ACTIVE")
	assert.Contains(t, out.String(), "*")
	assert.Contains(t, out.String(), "ready")
}

func TestRunStatusUsesUnknownForEmptyEmailAndAuthState(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)

	layout := paths.NewLayout(dir)
	registry := state.NewRegistry()
	acc := state.Account{
		Tag: "test",
	}
	registry.UpsertAccount(acc)
	err := state.SaveRegistry(layout.RegistryPath, registry)
	assert.NoError(t, err)

	cmd, out := newTestCommandOutput()

	err = runStatus(cmd, nil)
	assert.NoError(t, err)
	assert.Contains(t, out.String(), "unknown")
	assert.GreaterOrEqual(t, strings.Count(out.String(), "unknown"), 2)
}
