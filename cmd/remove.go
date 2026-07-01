package cmd

import (
	"github.com/rickyroynardson/codex-switch/internal/paths"
	"github.com/rickyroynardson/codex-switch/internal/state"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <tag>",
	Short: "Remove a Codex account with the given tag",
	Args:  cobra.ExactArgs(1),
	RunE:  runRemove,
}

func runRemove(cmd *cobra.Command, args []string) error {
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

	if err := registry.RemoveAccount(tag); err != nil {
		return err
	}

	if err := state.SaveRegistry(layout.RegistryPath, registry); err != nil {
		return err
	}

	cmd.Printf("removed account %s\n", tag)
	return nil
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
