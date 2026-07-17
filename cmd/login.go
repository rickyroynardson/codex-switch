package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/rickyroynardson/codex-switch/internal/codex"
	"github.com/rickyroynardson/codex-switch/internal/paths"
	"github.com/rickyroynardson/codex-switch/internal/state"
	"github.com/spf13/cobra"
)

var runCodexLogin = codex.RunLogin

var loginCmd = &cobra.Command{
	Use:   "login <tag>",
	Short: "Log in to a Codex account and associate it with a tag",
	Args:  cobra.ExactArgs(1),
	RunE:  runLogin,
}

func runLogin(cmd *cobra.Command, args []string) error {
	tag := args[0]
	if err := paths.ValidateTag(tag); err != nil {
		return err
	}

	layout, err := paths.DefaultLayout()
	if err != nil {
		return err
	}

	accountHome := layout.AccountDir(tag)
	authPath := layout.AccountAuthPath(tag)

	if err := ensureAccountHome(layout, tag); err != nil {
		return err
	}

	codexCommand, err := realCodexCommand(layout)
	if err != nil {
		return err
	}

	if err := runCodexLogin(codex.LoginOptions{CodexHome: accountHome, CodexCommand: codexCommand}); err != nil {
		return err
	}

	if _, err := os.Stat(authPath); err != nil {
		return fmt.Errorf("login completed but auth file was not found at %s: %w", authPath, err)
	}

	registry, err := state.LoadRegistry(layout.RegistryPath)
	if err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	existing, ok := registry.FindAccount(tag)

	createdAt := now
	if ok && existing.CreatedAt != "" {
		createdAt = existing.CreatedAt
	}

	email, err := codex.ReadEmailFromAuthFile(authPath)
	if err != nil {
		return err
	}

	registry.UpsertAccount(state.Account{
		Tag:       tag,
		AuthPath:  authPath,
		Email:     email,
		AuthState: state.AuthStateReady,
		CreatedAt: createdAt,
		UpdatedAt: now,
	})

	if err := state.SaveRegistry(layout.RegistryPath, registry); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "logged in account %s\n", tag)
	return nil
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
