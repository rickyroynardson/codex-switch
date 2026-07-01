package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/rickyroynardson/codex-switch/internal/paths"
	"github.com/rickyroynardson/codex-switch/internal/state"
	"github.com/spf13/cobra"
)

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

	for _, account := range registry.Accounts {
		active := ""
		if account.Tag == registry.ActiveTag {
			active = "*"
		}

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

	return w.Flush()
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
