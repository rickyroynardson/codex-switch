package cmd

import (
	"fmt"

	"github.com/rickyroynardson/codex-switch/internal/paths"
	"github.com/rickyroynardson/codex-switch/internal/state"
	"github.com/spf13/cobra"
)

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the current active account tag",
	RunE:  runCurrent,
}

func runCurrent(cmd *cobra.Command, args []string) error {
	layout, err := paths.DefaultLayout()
	if err != nil {
		return err
	}

	registry, err := state.LoadRegistry(layout.RegistryPath)
	if err != nil {
		return err
	}

	acc, ok := registry.ActiveAccount()
	if !ok {
		fmt.Fprintln(cmd.OutOrStdout(), "none")
		return nil
	}

	fmt.Fprintln(cmd.OutOrStdout(), acc.Tag)
	return nil
}

func init() {
	rootCmd.AddCommand(currentCmd)
}
