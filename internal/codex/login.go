package codex

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

type CommandRunner func(cmd string, args []string, env []string) error

type LoginOptions struct {
	CodexHome    string
	CodexCommand string
	Runner       CommandRunner
}

func RunLogin(opts LoginOptions) error {
	if opts.CodexHome == "" {
		return errors.New("codex home is required")
	}

	if err := EnsureFileAuthConfig(opts.CodexHome); err != nil {
		return err
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

	if err := runner(command, []string{"login"}, env); err != nil {
		return fmt.Errorf("codex login: %w", err)
	}

	return nil
}

func runCommand(command string, args []string, env []string) error {
	cmd := exec.Command(command, args...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runCommandQuiet(command string, args, env []string) error {
	cmd := exec.Command(command, args...)
	cmd.Env = env
	return cmd.Run()
}
