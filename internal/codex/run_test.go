package codex

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunWithHomeRequiresCodexHome(t *testing.T) {
	err := RunWithHome(RunOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "codex home is required")
}

func TestRunWithHomeRunsCodexWithHomeAndArgs(t *testing.T) {
	dir := t.TempDir()

	err := RunWithHome(RunOptions{
		CodexHome: dir,
		Args:      []string{"status"},
		Runner: func(cmd string, args, env []string) error {
			assert.Equal(t, "codex", cmd)
			assert.Equal(t, []string{"status"}, args)
			assert.Contains(t, env, "CODEX_HOME="+dir)
			return nil
		},
	})
	assert.NoError(t, err)
}

func TestRunWithHomeUsesCustomCodexCommand(t *testing.T) {
	dir := t.TempDir()

	err := RunWithHome(RunOptions{
		CodexHome:    dir,
		CodexCommand: "real-codex",
		Runner: func(cmd string, args, env []string) error {
			assert.Equal(t, "real-codex", cmd)
			assert.Contains(t, env, "CODEX_HOME="+dir)
			return nil
		},
	})
	assert.NoError(t, err)
}

func TestRunWithHomePropagatesRunnerError(t *testing.T) {
	dir := t.TempDir()

	err := RunWithHome(RunOptions{
		CodexHome: dir,
		Runner: func(cmd string, args, env []string) error {
			return assert.AnError
		},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "codex run")
}
