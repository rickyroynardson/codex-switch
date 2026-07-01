package cmd

import (
	"fmt"
	"os"

	"github.com/rickyroynardson/codex-switch/internal/paths"
	"github.com/rickyroynardson/codex-switch/internal/wrapper"
	"github.com/spf13/cobra"
)

var findRealCodex = wrapper.FindRealCodex
var installWrapper = wrapper.Install

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the Codex wrapper",
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	layout, err := paths.DefaultLayout()
	if err != nil {
		return err
	}

	realCodexPath, err := findRealCodex(layout.WrapperPath, os.Getenv("PATH"))
	if err != nil {
		return err
	}

	if err = installWrapper(layout.WrapperPath, realCodexPath, "codex-switch"); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "installed wrapper: %s\n", layout.WrapperPath)
	fmt.Fprintf(cmd.OutOrStdout(), "real codex: %s\n", realCodexPath)
	fmt.Fprintf(cmd.OutOrStdout(), "add to PATH: export PATH=%q:$PATH\n", layout.BinDir)

	return nil
}

func init() {
	rootCmd.AddCommand(initCmd)
}
