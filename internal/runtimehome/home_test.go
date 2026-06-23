package runtimehome

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rickyroynardson/codex-switch/internal/paths"
	"github.com/rickyroynardson/codex-switch/internal/state"
	"github.com/stretchr/testify/assert"
)

func TestIsSharedEntryReturnsTrueForAllowedEntries(t *testing.T) {
	cases := []string{"config.toml", "sessions", "skills", "plugins"}
	for _, name := range cases {
		if !IsSharedEntry(name) {
			t.Errorf("expecting %q to be shared entry", name)
		}
	}
}

func TestIsSharedEntryReturnsFalseForSensitiveOrUnknownEntries(t *testing.T) {
	cases := []string{"auth.json", "mcp_oauth.json", "log", "unknown"}
	for _, name := range cases {
		if IsSharedEntry(name) {
			t.Errorf("expecting %q to not be shared entry", name)
		}
	}
}

func writeAuthFile(t *testing.T, authPath, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(authPath), 0700); err != nil {
		t.Fatalf("create auth dir: %v", err)
	}

	if err := os.WriteFile(authPath, []byte(contents), 0600); err != nil {
		t.Fatalf("write auth file: %v", err)
	}
}

func TestAssembleCopiesAuthFile(t *testing.T) {
	layout := paths.NewLayout(t.TempDir())
	authPath := layout.AccountAuthPath("work")
	writeAuthFile(t, authPath, `{"token":"work"}`)

	account := state.Account{
		Tag:      "work",
		AuthPath: authPath,
	}

	err := Assemble(layout, account)
	assert.NoError(t, err)

	b, err := os.ReadFile(filepath.Join(layout.CurrentHomeDir, "auth.json"))
	assert.NoError(t, err)
	assert.Equal(t, `{"token":"work"}`, string(b))
}

func TestAssembleLinksSharedEntries(t *testing.T) {
	layout := paths.NewLayout(t.TempDir())
	authPath := layout.AccountAuthPath("work")
	writeAuthFile(t, authPath, `{}`)

	if err := os.MkdirAll(layout.SharedDir, 0700); err != nil {
		t.Fatalf("created shared dir: %v", err)
	}

	sharedConfig := filepath.Join(layout.SharedDir, "config.toml")
	if err := os.WriteFile(sharedConfig, []byte("shared config"), 0600); err != nil {
		t.Fatalf("write shared config: %v", err)
	}

	sharedSkills := filepath.Join(layout.SharedDir, "skills")
	if err := os.MkdirAll(sharedSkills, 0700); err != nil {
		t.Fatalf("create shared skills: %v", err)
	}

	account := state.Account{
		Tag:      "work",
		AuthPath: authPath,
	}

	err := Assemble(layout, account)
	assert.NoError(t, err)

	configLink := filepath.Join(layout.CurrentHomeDir, "config.toml")
	configInfo, err := os.Lstat(configLink)
	assert.NoError(t, err)
	assert.NotZero(t, configInfo.Mode()&os.ModeSymlink)

	configTarget, err := os.Readlink(configLink)
	assert.NoError(t, err)
	assert.Equal(t, sharedConfig, configTarget)

	skillsLink := filepath.Join(layout.CurrentHomeDir, "skills")
	skillsInfo, err := os.Lstat(skillsLink)
	assert.NoError(t, err)
	assert.NotZero(t, skillsInfo.Mode()&os.ModeSymlink)

	skillsTarget, err := os.Readlink(skillsLink)
	assert.NoError(t, err)
	assert.Equal(t, sharedSkills, skillsTarget)
}

func TestAssembleSkipsMissingSharedEntries(t *testing.T) {
	layout := paths.NewLayout(t.TempDir())
	authPath := layout.AccountAuthPath("work")
	writeAuthFile(t, authPath, `{}`)

	account := state.Account{
		Tag:      "work",
		AuthPath: authPath,
	}

	err := Assemble(layout, account)
	assert.NoError(t, err)

	_, err = os.Lstat(filepath.Join(layout.CurrentHomeDir, "auth.json"))
	assert.NoError(t, err)

	_, err = os.Lstat(filepath.Join(layout.CurrentHomeDir, "config.toml"))
	assert.True(t, os.IsNotExist(err))
}

func TestAssembleReplacesExistingCurrentHome(t *testing.T) {
	layout := paths.NewLayout(t.TempDir())
	authPath := layout.AccountAuthPath("work")
	writeAuthFile(t, authPath, `{"token":"new"}`)

	if err := os.MkdirAll(layout.CurrentHomeDir, 0700); err != nil {
		t.Fatalf("create existing current home: %v", err)
	}

	stalePath := filepath.Join(layout.CurrentHomeDir, "stale.txt")
	if err := os.WriteFile(stalePath, []byte("stale"), 0600); err != nil {
		t.Fatalf("write stale file: %v", err)
	}

	account := state.Account{
		Tag:      "work",
		AuthPath: authPath,
	}

	err := Assemble(layout, account)
	assert.NoError(t, err)

	_, err = os.Stat(stalePath)
	assert.True(t, os.IsNotExist(err))

	b, err := os.ReadFile(filepath.Join(layout.CurrentHomeDir, "auth.json"))
	assert.NoError(t, err)
	assert.Equal(t, `{"token":"new"}`, string(b))
}

func TestAssembleReturnsErrorWhenAuthMissing(t *testing.T) {
	layout := paths.NewLayout(t.TempDir())

	account := state.Account{
		Tag:      "work",
		AuthPath: layout.AccountAuthPath("work"),
	}

	err := Assemble(layout, account)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "account auth file not found")
}
