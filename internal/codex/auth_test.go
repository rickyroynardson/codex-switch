package codex

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnsureFileAuthConfigCreatesHome(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "codex-home")

	err := EnsureFileAuthConfig(dir)
	assert.NoError(t, err)

	info, err := os.Stat(dir)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestEnsureFileAuthConfigWritesConfig(t *testing.T) {
	dir := t.TempDir()

	err := EnsureFileAuthConfig(dir)
	assert.NoError(t, err)

	b, err := os.ReadFile(filepath.Join(dir, "config.toml"))
	assert.NoError(t, err)
	assert.Equal(t, FileAuthConfig, string(b))
}

func TestEnsureFileAuthConfigOverwritesExistingConfig(t *testing.T) {
	dir := t.TempDir()

	err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("existing config"), 0600)
	assert.NoError(t, err)

	err = EnsureFileAuthConfig(dir)
	assert.NoError(t, err)

	b, err := os.ReadFile(filepath.Join(dir, "config.toml"))
	assert.NoError(t, err)
	assert.Equal(t, FileAuthConfig, string(b))
}
