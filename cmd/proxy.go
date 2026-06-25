/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"errors"

	"github.com/rickyroynardson/codex-switch/internal/codex"
	"github.com/rickyroynardson/codex-switch/internal/paths"
	"github.com/rickyroynardson/codex-switch/internal/runtimehome"
	"github.com/rickyroynardson/codex-switch/internal/state"
	"github.com/spf13/cobra"
)

var (
	assembleRuntimeHome = runtimehome.Assemble
	runCodexWithHome    = codex.RunWithHome
)

// proxyCmd represents the proxy command
var proxyCmd = &cobra.Command{
	Use:   "proxy -- [codex args...]",
	Short: "Run Codex with the active switched account",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args:               cobra.ArbitraryArgs,
	DisableFlagParsing: true,
	RunE:               runProxy,
}

func runProxy(cmd *cobra.Command, args []string) error {
	layout, err := paths.DefaultLayout()
	if err != nil {
		return err
	}

	registry, err := state.LoadRegistry(layout.RegistryPath)
	if err != nil {
		return err
	}

	account, ok := registry.ActiveAccount()
	if !ok {
		return errors.New("no active account")
	}

	if err := assembleRuntimeHome(layout, account); err != nil {
		return err
	}

	return runCodexWithHome(codex.RunOptions{
		CodexHome: layout.CurrentHomeDir,
		Args:      args,
	})
}

func init() {
	rootCmd.AddCommand(proxyCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// proxyCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// proxyCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
