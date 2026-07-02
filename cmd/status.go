package cmd

import (
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/rickyroynardson/codex-switch/internal/codex"
	"github.com/rickyroynardson/codex-switch/internal/paths"
	"github.com/rickyroynardson/codex-switch/internal/state"
	"github.com/spf13/cobra"
)

var checkCodexLoginStatus = codex.CheckLoginStatus

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of all accounts",
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	layout, err := paths.DefaultLayout()
	if err != nil {
		return err
	}

	registry, err := state.LoadRegistry(layout.RegistryPath)
	if err != nil {
		return err
	}

	if len(registry.Accounts) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "no accounts")
		return nil
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ACTIVE\tTAG\tAUTH\tACCOUNT")

	for i, account := range registry.Accounts {
		active := ""
		if account.Tag == registry.ActiveTag {
			active = "*"
		}

		if account.AuthPath == "" {
			account.AuthPath = layout.AccountAuthPath(account.Tag)
		}

		ready, err := checkCodexLoginStatus(codex.LoginStatusOptions{
			CodexHome: layout.AccountDir(account.Tag),
		})
		if err != nil {
			return err
		}

		if ready {
			account.AuthState = state.AuthStateReady

			if email, err := codex.ReadEmailFromAuthFile(account.AuthPath); err == nil && email != "" {
				account.Email = email
			}
		} else {
			account.AuthState = state.AuthStateNeedsLogin
		}
		account.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

		registry.Accounts[i] = account

		email := account.Email
		if email == "" {
			email = "unknown"
		}

		authState := account.AuthState
		if authState == "" {
			authState = state.AuthStateUnknown
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", active, account.Tag, authState, email)
	}

	if err := w.Flush(); err != nil {
		return err
	}

	return state.SaveRegistry(layout.RegistryPath, registry)
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
