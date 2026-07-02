package codex

import (
	"errors"
	"os"
)

type LoginStatusOptions struct {
	CodexHome    string
	CodexCommand string
	Runner       CommandRunner
}

func CheckLoginStatus(opts LoginStatusOptions) (bool, error) {
	if opts.CodexHome == "" {
		return false, errors.New("codex home is required")
	}

	if err := EnsureFileAuthConfig(opts.CodexHome); err != nil {
		return false, err
	}

	command := opts.CodexCommand
	if command == "" {
		command = "codex"
	}

	runner := opts.Runner
	if runner == nil {
		runner = runCommandQuiet
	}

	env := append(os.Environ(), "CODEX_HOME="+opts.CodexHome)

	if err := runner(command, []string{"login", "status"}, env); err != nil {
		return false, err
	}

	return true, nil
}
