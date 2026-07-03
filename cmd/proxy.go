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

const EnvRealCodex = "CODEX_SWITCH_REAL_CODEX"

var proxyCmd = &cobra.Command{
	Use:                "proxy -- [codex args...]",
	Short:              "Run Codex with the active switched account",
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

	codexCommand, err := realCodexCommand(layout)
	if err != nil {
		return err
	}

	return runCodexWithHome(codex.RunOptions{
		CodexHome:    layout.CurrentHomeDir,
		CodexCommand: codexCommand,
		Args:         args,
	})
}

func init() {
	rootCmd.AddCommand(proxyCmd)
}
