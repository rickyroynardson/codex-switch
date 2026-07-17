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

var (
	checkCodexLoginStatus   = codex.CheckLoginStatus
	probeCodexAccountLimits = codex.ProbeAccountLimits
)

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
	refresh := statusRefreshFromCommand(cmd)

	if len(registry.Accounts) == 0 && tag == "" {
		fmt.Fprintln(cmd.OutOrStdout(), "no accounts")
		return nil
	}

	// Only the refresh path spawns codex, so a plain cache read needs neither
	// the binary on PATH nor a registry write.
	codexCommand := ""
	if refresh {
		codexCommand, err = realCodexCommand(layout)
		if err != nil {
			return err
		}
	}

	now := time.Now()

	if tag != "" {
		for i, account := range registry.Accounts {
			if account.Tag != tag {
				continue
			}

			snapshot := cachedRateLimitSnapshot(account)
			if refresh {
				account, snapshot, err = refreshAccountStatus(layout, account, codexCommand)
				if err != nil {
					return err
				}

				registry.Accounts[i] = account
				if err := state.SaveRegistry(layout.RegistryPath, registry); err != nil {
					return err
				}
			}

			printAccountStatusDetail(cmd, layout, registry, account, snapshot)
			return nil
		}

		return fmt.Errorf("unknown account tag: %s", tag)
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ACTIVE\tTAG\t5H_LEFT\tWEEKLY_LEFT\t5H_RESET\tWEEKLY_RESET\tAUTH\tACCOUNT\tCHECKED")

	changed := false
	for i, account := range registry.Accounts {
		active := ""
		if account.Tag == registry.ActiveTag {
			active = "*"
		}

		snapshot := cachedRateLimitSnapshot(account)
		if refresh {
			// ponytail: probes run serially; parallelize across accounts if
			// anyone runs --refresh with enough accounts to feel it.
			account, snapshot, err = refreshAccountStatus(layout, account, codexCommand)
			if err != nil {
				return err
			}
			registry.Accounts[i] = account
			changed = true
		}

		authState := account.AuthState
		if authState == "" {
			authState = state.AuthStateUnknown
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			active,
			account.Tag,
			formatRemainingPercent(snapshot.FiveHourUsedPct),
			formatRemainingPercent(snapshot.WeeklyUsedPct),
			formatStatusValue(snapshot.FiveHourResetIn),
			formatStatusValue(snapshot.WeeklyResetIn),
			authState,
			formatStatusValue(account.Email),
			formatCheckedAge(account.LastStatusCheckAt, now),
		)
	}

	if err := w.Flush(); err != nil {
		return err
	}

	if changed {
		return state.SaveRegistry(layout.RegistryPath, registry)
	}

	return nil
}

func init() {
	statusCmd.Flags().StringP("tag", "t", "", "Show status for a specific account tag")
	statusCmd.Flags().Bool("refresh", false, "Probe Codex for live auth and quota instead of showing cached values")
	rootCmd.AddCommand(statusCmd)
}

func statusTagFromCommand(cmd *cobra.Command) string {
	flag := cmd.Flags().Lookup("tag")
	if flag == nil {
		return ""
	}

	return flag.Value.String()
}

func statusRefreshFromCommand(cmd *cobra.Command) bool {
	flag := cmd.Flags().Lookup("refresh")
	if flag == nil {
		return false
	}

	return flag.Value.String() == "true"
}

// cachedRateLimitSnapshot rebuilds a display snapshot from the account's last
// persisted status, or an "never checked" placeholder when none exists.
func cachedRateLimitSnapshot(account state.Account) codex.RateLimitSnapshot {
	s := account.LastKnownStatus
	if s == nil {
		return codex.UnknownRateLimitSnapshot("never")
	}

	return codex.RateLimitSnapshot{
		FiveHourUsedPct: s.FiveHourUsedPct,
		WeeklyUsedPct:   s.WeeklyUsedPct,
		FiveHourResetIn: s.FiveHourResetIn,
		WeeklyResetIn:   s.WeeklyResetIn,
		RawLimitSource:  s.RawLimitSource,
		PlanType:        s.PlanType,
	}
}

func formatCheckedAge(ts string, now time.Time) string {
	if ts == "" {
		return "never"
	}

	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return "unknown"
	}

	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours())/24)
	}
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

	checkedAt := time.Now().UTC().Format(time.RFC3339)

	account.UpdatedAt = checkedAt
	account.LastStatusCheckAt = checkedAt
	account.LastKnownStatus = nil

	snapshot := codex.UnknownRateLimitSnapshot("unknown")
	if account.AuthState == state.AuthStateReady {
		snapshot = probeCodexAccountLimits(codex.ProbeAccountLimitsOptions{
			CodexHome:    layout.AccountDir(account.Tag),
			CodexCommand: codexCommand,
		})

		account.LastKnownStatus = registryStatusSnapshot(snapshot)
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
	fmt.Fprintf(cmd.OutOrStdout(), "last_switch_at: %s\n", formatStatusValue(account.LastSwitchAt))
	fmt.Fprintf(cmd.OutOrStdout(), "last_status_check_at: %s\n", formatStatusValue(account.LastStatusCheckAt))
	fmt.Fprintf(cmd.OutOrStdout(), "five_hour_left_pct: %s\n", formatRemainingPercent(snapshot.FiveHourUsedPct))
	fmt.Fprintf(cmd.OutOrStdout(), "weekly_left_pct: %s\n", formatRemainingPercent(snapshot.WeeklyUsedPct))
	fmt.Fprintf(cmd.OutOrStdout(), "five_hour_reset_in: %s\n", formatStatusValue(snapshot.FiveHourResetIn))
	fmt.Fprintf(cmd.OutOrStdout(), "weekly_reset_in: %s\n", formatStatusValue(snapshot.WeeklyResetIn))
	fmt.Fprintf(cmd.OutOrStdout(), "raw_limit_source: %s\n", formatStatusValue(snapshot.RawLimitSource))
	fmt.Fprintf(cmd.OutOrStdout(), "account: %s\n", formatStatusValue(account.Email))
	fmt.Fprintf(cmd.OutOrStdout(), "auth_state: %s\n", authState)
	fmt.Fprintf(cmd.OutOrStdout(), "auth_storage_path: %s\n", account.AuthPath)
}

func registryStatusSnapshot(snapshot codex.RateLimitSnapshot) *state.StatusSnapshot {
	return &state.StatusSnapshot{
		FiveHourUsedPct: snapshot.FiveHourUsedPct,
		WeeklyUsedPct:   snapshot.WeeklyUsedPct,
		FiveHourResetIn: snapshot.FiveHourResetIn,
		WeeklyResetIn:   snapshot.WeeklyResetIn,
		RawLimitSource:  snapshot.RawLimitSource,
		PlanType:        snapshot.PlanType,
	}
}
