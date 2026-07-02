package codex

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckLoginStatusRequiresCodexHome(t *testing.T) {
	ready, err := CheckLoginStatus(LoginStatusOptions{})

	require.Error(t, err)
	assert.False(t, ready)
	assert.Contains(t, err.Error(), "codex home is required")
}

func TestCheckLoginStatusRunsCodexLoginStatusWithCodexHome(t *testing.T) {
	dir := t.TempDir()

	ready, err := CheckLoginStatus(LoginStatusOptions{
		CodexHome: dir,
		Runner: func(cmd string, args, env []string) error {
			assert.Equal(t, "codex", cmd)
			assert.Equal(t, []string{"login", "status"}, args)
			assert.Contains(t, env, "CODEX_HOME="+dir)
			return nil
		},
	})

	require.NoError(t, err)
	assert.True(t, ready)
}

func TestCheckLoginStatusRunsCodexLoginWithCustomCommand(t *testing.T) {
	dir := t.TempDir()

	ready, err := CheckLoginStatus(LoginStatusOptions{
		CodexHome:    dir,
		CodexCommand: "codex-switch",
		Runner: func(cmd string, args, env []string) error {
			assert.Equal(t, "codex-switch", cmd)
			return nil
		},
	})

	require.NoError(t, err)
	assert.True(t, ready)
}

func TestCheckLoginStatusReturnsFalseWhenCodexStatusFails(t *testing.T) {
	dir := t.TempDir()

	ready, err := CheckLoginStatus(LoginStatusOptions{
		CodexHome: dir,
		Runner: func(cmd string, args, env []string) error {
			return assert.AnError
		},
	})

	require.Error(t, err)
	assert.False(t, ready)
}
