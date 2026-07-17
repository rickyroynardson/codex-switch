package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rickyroynardson/codex-switch/internal/accounthome"
	"github.com/rickyroynardson/codex-switch/internal/paths"
	"github.com/rickyroynardson/codex-switch/internal/wrapper"
	"github.com/spf13/cobra"
)

var findRealCodex = wrapper.FindRealCodex
var installWrapper = wrapper.Install
var importSharedState = accounthome.ImportSharedState
var userHomeDir = os.UserHomeDir

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

	importMessage := ""
	empty, err := sharedStateEmpty(layout)
	if err != nil {
		return err
	}

	if empty {
		home, err := userHomeDir()
		if err != nil {
			return err
		}

		sourceHome := filepath.Join(home, ".codex")
		if _, err := os.Stat(sourceHome); err == nil {
			if err := importSharedState(layout, sourceHome); err != nil {
				return err
			}
			importMessage = fmt.Sprintf("imported shared state from: %s\n", sourceHome)
		} else if !os.IsNotExist(err) {
			return err
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "installed wrapper: %s\n", layout.WrapperPath)
	fmt.Fprintf(cmd.OutOrStdout(), "real codex: %s\n", realCodexPath)
	fmt.Fprintf(cmd.OutOrStdout(), "add to PATH: export PATH=%q:$PATH\n", layout.BinDir)
	if importMessage != "" {
		fmt.Fprint(cmd.OutOrStdout(), importMessage)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func sharedStateEmpty(layout paths.Layout) (bool, error) {
	entries, err := os.ReadDir(layout.SharedDir)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}

	return len(entries) == 0, nil
}
