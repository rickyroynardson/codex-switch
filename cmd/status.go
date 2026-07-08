package cmd

import (
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/rickyroynardson/codex-switch/internal/codex"
	"github.com/rickyroynardson/codex-switch/internal/paths"
	"github.com/rickyroynardson/codex-switch/internal/state"
	"github.com/spf13/cobra"
)

var checkCodexLoginStatus = codex.CheckLoginStatus
var probeCodexAccountLimits = codex.ProbeAccountLimits

func formatRemainingPercent(used *int) string {
	if used == nil {
		return "unknown"
	}

	remaining := 100 - *used
	if remaining < 0 {
		remaining = 0
	}

	return fmt.Sprintf("%d%%", remaining)
}

func formatStatusValue(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of all accounts",
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	layout, err := paths.DefaultLayout()
	if err != nil {
		return err
	}

	registry, err := state.LoadRegistry(layout.RegistryPath)
	if err != nil {
		return err
	}

	tag := statusTagFromCommand(cmd)

	if len(registry.Accounts) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "no accounts")
		return nil
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ACTIVE\tTAG\t5H_LEFT\tWEEKLY_LEFT\t5H_RESET\tWEEKLY_RESET\tAUTH\tACCOUNT")

	codexCommand, err := realCodexCommand(layout)
	if err != nil {
		return err
	}

	if tag != "" {
		for i, account := range registry.Accounts {
			if account.Tag != tag {
				continue
			}

			account, snapshot, err := refreshAccountStatus(layout, account, codexCommand)
			if err != nil {
				return err
			}

			registry.Accounts[i] = account
			if err := state.SaveRegistry(layout.RegistryPath, registry); err != nil {
				return err
			}

			printAccountStatusDetail(cmd, layout, registry, account, snapshot)
			return nil
		}

		return fmt.Errorf("unknown account tag: %s", tag)
	}

	for i, account := range registry.Accounts {
		active := ""
		if account.Tag == registry.ActiveTag {
			active = "*"
		}

		if account.AuthPath == "" {
			account.AuthPath = layout.AccountAuthPath(account.Tag)
		}

		ready, err := checkCodexLoginStatus(codex.LoginStatusOptions{
			CodexHome:    layout.AccountDir(account.Tag),
			CodexCommand: codexCommand,
		})
		if err != nil {
			return err
		}

		if ready {
			account.AuthState = state.AuthStateReady

			if email, err := codex.ReadEmailFromAuthFile(account.AuthPath); err == nil && email != "" {
				account.Email = email
			}
		} else {
			account.AuthState = state.AuthStateNeedsLogin
		}
		account.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

		registry.Accounts[i] = account

		email := account.Email
		if email == "" {
			email = "unknown"
		}

		authState := account.AuthState
		if authState == "" {
			authState = state.AuthStateUnknown
		}

		snapshot := codex.UnknownRateLimitSnapshot("unknown")
		if account.AuthState == state.AuthStateReady {
			snapshot = probeCodexAccountLimits(codex.ProbeAccountLimitsOptions{
				CodexHome:    layout.AccountDir(account.Tag),
				CodexCommand: codexCommand,
			})
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			active,
			account.Tag,
			formatRemainingPercent(snapshot.FiveHourUsedPct),
			formatRemainingPercent(snapshot.WeeklyUsedPct),
			formatStatusValue(snapshot.FiveHourResetIn),
			formatStatusValue(snapshot.WeeklyResetIn),
			authState,
			email,
		)
	}

	if err := w.Flush(); err != nil {
		return err
	}

	return state.SaveRegistry(layout.RegistryPath, registry)
}

func init() {
	statusCmd.Flags().StringP("tag", "t", "", "Show status for a specific account tag")
	rootCmd.AddCommand(statusCmd)
}

func statusTagFromCommand(cmd *cobra.Command) string {
	flag := cmd.Flags().Lookup("tag")
	if flag == nil {
		return ""
	}

	return flag.Value.String()
}

func refreshAccountStatus(layout paths.Layout, account state.Account, codexCommand string) (state.Account, codex.RateLimitSnapshot, error) {
	if account.AuthPath == "" {
		account.AuthPath = layout.AccountAuthPath(account.Tag)
	}

	ready, err := checkCodexLoginStatus(codex.LoginStatusOptions{
		CodexHome:    layout.AccountDir(account.Tag),
		CodexCommand: codexCommand,
	})
	if err != nil {
		return account, codex.RateLimitSnapshot{}, err
	}

	if ready {
		account.AuthState = state.AuthStateReady

		if email, err := codex.ReadEmailFromAuthFile(account.AuthPath); err == nil && email != "" {
			account.Email = email
		}
	} else {
		account.AuthState = state.AuthStateNeedsLogin
	}

	account.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	snapshot := codex.UnknownRateLimitSnapshot("unknown")
	if account.AuthState == state.AuthStateReady {
		snapshot = probeCodexAccountLimits(codex.ProbeAccountLimitsOptions{
			CodexHome:    layout.AccountDir(account.Tag),
			CodexCommand: codexCommand,
		})
	}

	return account, snapshot, nil
}

func printAccountStatusDetail(cmd *cobra.Command, layout paths.Layout, registry state.Registry, account state.Account, snapshot codex.RateLimitSnapshot) {
	active := "no"
	if account.Tag == registry.ActiveTag {
		active = "yes"
	}

	authState := account.AuthState
	if authState == "" {
		authState = state.AuthStateUnknown
	}

	fmt.Fprintf(cmd.OutOrStdout(), "tag: %s\n", account.Tag)
	fmt.Fprintf(cmd.OutOrStdout(), "active: %s\n", active)
	fmt.Fprintf(cmd.OutOrStdout(), "five_hour_left_pct: %s\n", formatRemainingPercent(snapshot.FiveHourUsedPct))
	fmt.Fprintf(cmd.OutOrStdout(), "weekly_left_pct: %s\n", formatRemainingPercent(snapshot.WeeklyUsedPct))
	fmt.Fprintf(cmd.OutOrStdout(), "five_hour_reset_in: %s\n", formatStatusValue(snapshot.FiveHourResetIn))
	fmt.Fprintf(cmd.OutOrStdout(), "weekly_reset_in: %s\n", formatStatusValue(snapshot.WeeklyResetIn))
	fmt.Fprintf(cmd.OutOrStdout(), "raw_limit_source: %s\n", formatStatusValue(snapshot.RawLimitSource))
	fmt.Fprintf(cmd.OutOrStdout(), "account: %s\n", formatStatusValue(account.Email))
	fmt.Fprintf(cmd.OutOrStdout(), "auth_state: %s\n", authState)
	fmt.Fprintf(cmd.OutOrStdout(), "auth_storage_path: %s\n", account.AuthPath)
}
