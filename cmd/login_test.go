package cmd

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rickyroynardson/codex-switch/internal/codex"
	"github.com/rickyroynardson/codex-switch/internal/paths"
	"github.com/rickyroynardson/codex-switch/internal/state"
	"github.com/stretchr/testify/assert"
)

func TestRunLoginRegistersAccount(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)

	oldRunCodexLogin := runCodexLogin
	t.Cleanup(func() {
		runCodexLogin = oldRunCodexLogin
	})

	runCodexLogin = func(opts codex.LoginOptions) error {
		if err := os.MkdirAll(opts.CodexHome, 0700); err != nil {
			return err
		}

		authPath := filepath.Join(opts.CodexHome, "auth.json")

		header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
		payload := base64.RawURLEncoding.EncodeToString([]byte(`{"email": "work@mail.com"}`))
		idToken := header + "." + payload + "."
		authJSON := `{"tokens": {"id_token": "` + idToken + `"}}`
		return os.WriteFile(authPath, []byte(authJSON), 0600)
	}

	cmd, out := newTestCommandOutput()

	err := runLogin(cmd, []string{"work"})
	assert.NoError(t, err)
	assert.Equal(t, "logged in account work\n", out.String())

	layout := paths.NewLayout(dir)
	registry, err := state.LoadRegistry(layout.RegistryPath)
	assert.NoError(t, err)

	account, ok := registry.FindAccount("work")
	assert.True(t, ok)
	assert.Equal(t, "work", registry.ActiveTag)
	assert.Equal(t, "work", account.Tag)
	assert.Equal(t, "work@mail.com", account.Email)
	assert.Equal(t, layout.AccountAuthPath("work"), account.AuthPath)
	assert.Equal(t, state.AuthStateReady, account.AuthState)
	assert.NotEmpty(t, account.CreatedAt)
	assert.NotEmpty(t, account.UpdatedAt)
}

func TestRunLoginReturnsErrorForInvalidTag(t *testing.T) {
	oldRunCodexLogin := runCodexLogin
	t.Cleanup(func() {
		runCodexLogin = oldRunCodexLogin
	})

	runCodexLogin = func(opts codex.LoginOptions) error {
		t.Fatalf("runCodexLogin should not be called")
		return nil
	}

	cmd, _ := newTestCommandOutput()

	err := runLogin(cmd, []string{"../bad"})
	assert.Error(t, err)
}

func TestRunLoginReturnsErrorWhenCodexLoginFails(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)

	oldRunCodexLogin := runCodexLogin
	t.Cleanup(func() {
		runCodexLogin = oldRunCodexLogin
	})

	runCodexLogin = func(opts codex.LoginOptions) error {
		return errors.New("boom")
	}

	cmd, _ := newTestCommandOutput()

	err := runLogin(cmd, []string{"work"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "boom")

	layout := paths.NewLayout(dir)
	registry, err := state.LoadRegistry(layout.RegistryPath)
	assert.NoError(t, err)
	assert.Len(t, registry.Accounts, 0)
}

func TestRunLoginReturnsErrorWhenAuthFileMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)

	oldRunCodexLogin := runCodexLogin
	t.Cleanup(func() {
		runCodexLogin = oldRunCodexLogin
	})

	runCodexLogin = func(opts codex.LoginOptions) error {
		return nil
	}

	cmd, _ := newTestCommandOutput()

	err := runLogin(cmd, []string{"work"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "auth file")

	layout := paths.NewLayout(dir)
	registry, err := state.LoadRegistry(layout.RegistryPath)
	assert.NoError(t, err)
	assert.Len(t, registry.Accounts, 0)
}
