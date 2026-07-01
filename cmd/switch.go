package cmd

import (
	"fmt"

	"github.com/rickyroynardson/codex-switch/internal/paths"
	"github.com/rickyroynardson/codex-switch/internal/state"
	"github.com/spf13/cobra"
)

var switchCmd = &cobra.Command{
	Use:   "switch <tag>",
	Short: "Switch the active Codex account",
	Args:  cobra.ExactArgs(1),
	RunE:  runSwitch,
}

func runSwitch(cmd *cobra.Command, args []string) error {
	tag := args[0]
	if err := paths.ValidateTag(tag); err != nil {
		return err
	}

	layout, err := paths.DefaultLayout()
	if err != nil {
		return err
	}

	registry, err := state.LoadRegistry(layout.RegistryPath)
	if err != nil {
		return err
	}

	if err := registry.SetActiveTag(tag); err != nil {
		return err
	}
	if err := state.SaveRegistry(layout.RegistryPath, registry); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "switched to %s\n", tag)
	return nil
}

func init() {
	rootCmd.AddCommand(switchCmd)
}
