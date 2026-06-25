package codex

import (
	"errors"
	"fmt"
	"os"
)

type RunOptions struct {
	CodexHome    string
	CodexCommand string
	Args         []string
	Runner       CommandRunner
}

func RunWithHome(opts RunOptions) error {
	if opts.CodexHome == "" {
		return errors.New("codex home is required")
	}

	command := opts.CodexCommand
	if command == "" {
		command = "codex"
	}

	runner := opts.Runner
	if runner == nil {
		runner = runCommand
	}

	env := append(os.Environ(), "CODEX_HOME="+opts.CodexHome)

	if err := runner(command, opts.Args, env); err != nil {
		return fmt.Errorf("codex run: %w", err)
	}

	return nil
}
