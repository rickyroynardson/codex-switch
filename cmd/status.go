/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/rickyroynardson/codex-switch/internal/paths"
	"github.com/rickyroynardson/codex-switch/internal/state"
	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: runStatus,
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

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// statusCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// statusCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
