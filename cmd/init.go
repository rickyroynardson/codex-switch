/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
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

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Install the Codex wrapper",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: runInit,
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

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// initCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// initCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
