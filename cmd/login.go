/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
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

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login <tag>",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args: cobra.ExactArgs(1),
	RunE: runLogin,
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

	if err := runCodexLogin(codex.LoginOptions{CodexHome: accountHome}); err != nil {
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

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// loginCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// loginCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
