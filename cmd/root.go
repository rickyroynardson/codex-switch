package cmd

import (
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "codex-switch",
	Short:   "Codex Switch is a tool for managing multiple Codex accounts and switching between them.",
	Version: buildVersion(),
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// buildVersion reports the module version stamped by `go install`; local
// `go build` binaries carry no version, so they report "(devel)".
func buildVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "(devel)"
}
