/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/rickyroynardson/codex-switch/internal/paths"
	"github.com/rickyroynardson/codex-switch/internal/state"
	"github.com/spf13/cobra"
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:   "remove <tag>",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args: cobra.ExactArgs(1),
	RunE: runRemove,
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

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// removeCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// removeCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
