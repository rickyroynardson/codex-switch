package codex

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunLoginRequiresCodexHome(t *testing.T) {
	opts := LoginOptions{}
	err := RunLogin(opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "codex home is required")
}

func TestRunLoginEnsuresFileAuthConfig(t *testing.T) {
	opts := LoginOptions{
		CodexHome: t.TempDir(),
		Runner: func(cmd string, args []string, env []string) error {
			return nil
		},
	}
	err := RunLogin(opts)
	assert.NoError(t, err)

	b, err := os.ReadFile(filepath.Join(opts.CodexHome, "config.toml"))
	assert.NoError(t, err)
	assert.Equal(t, FileAuthConfig, string(b))
}

func TestRunLoginRunsCodexLoginWithCodexHome(t *testing.T) {
	dir := t.TempDir()
	opts := LoginOptions{
		CodexHome: dir,
		Runner: func(cmd string, args []string, env []string) error {
			assert.Equal(t, "codex", cmd)
			assert.Equal(t, []string{"login"}, args)
			assert.Contains(t, env, "CODEX_HOME="+dir)
			return nil
		},
	}
	err := RunLogin(opts)
	assert.NoError(t, err)
}

func TestRunLoginPropagatesRunnerError(t *testing.T) {
	dir := t.TempDir()
	opts := LoginOptions{
		CodexHome: dir,
		Runner: func(cmd string, args []string, env []string) error {
			return assert.AnError
		},
	}
	err := RunLogin(opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "codex login")
}

func TestRunLoginUsesCustomCodexCommand(t *testing.T) {
	dir := t.TempDir()
	opts := LoginOptions{
		CodexHome:    dir,
		CodexCommand: "real-codex",
		Runner: func(cmd string, args []string, env []string) error {
			assert.Equal(t, "real-codex", cmd)
			return nil
		},
	}
	err := RunLogin(opts)
	assert.NoError(t, err)
}
