package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/rickyroynardson/codex-switch/internal/codex"
	"github.com/rickyroynardson/codex-switch/internal/paths"
	"github.com/rickyroynardson/codex-switch/internal/state"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCommandOutput() (*cobra.Command, *bytes.Buffer) {
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	return cmd, &out
}

// addRefreshFlag registers --refresh=true so a test exercises the live probe path.
func addRefreshFlag(cmd *cobra.Command) {
	cmd.Flags().Bool("refresh", true, "")
}

func stubCheckCodexLoginStatus(t *testing.T, ready bool) {
	t.Helper()

	old := checkCodexLoginStatus
	t.Cleanup(func() {
		checkCodexLoginStatus = old
	})

	checkCodexLoginStatus = func(opts codex.LoginStatusOptions) (bool, error) {
		return ready, nil
	}
}

func stubProbeCodexAccountLimits(t *testing.T, snapshot codex.RateLimitSnapshot) {
	t.Helper()

	old := probeCodexAccountLimits
	t.Cleanup(func() {
		probeCodexAccountLimits = old
	})

	probeCodexAccountLimits = func(opts codex.ProbeAccountLimitsOptions) codex.RateLimitSnapshot {
		return snapshot
	}
}

func failIfProbed(t *testing.T) {
	t.Helper()

	oldLogin := checkCodexLoginStatus
	oldProbe := probeCodexAccountLimits
	t.Cleanup(func() {
		checkCodexLoginStatus = oldLogin
		probeCodexAccountLimits = oldProbe
	})

	checkCodexLoginStatus = func(opts codex.LoginStatusOptions) (bool, error) {
		t.Fatalf("checkCodexLoginStatus should not be called without --refresh")
		return false, nil
	}
	probeCodexAccountLimits = func(opts codex.ProbeAccountLimitsOptions) codex.RateLimitSnapshot {
		t.Fatalf("probeCodexAccountLimits should not be called without --refresh")
		return codex.RateLimitSnapshot{}
	}
}

func TestRunStatusPrintsNoAccounts(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)

	cmd, out := newTestCommandOutput()

	err := runStatus(cmd, nil)
	assert.NoError(t, err)
	assert.Equal(t, "no accounts\n", out.String())
}

func TestRunStatusDefaultShowsCachedWithoutProbing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)
	failIfProbed(t)

	fiveHourUsed := 25
	weeklyUsed := 50

	layout := paths.NewLayout(dir)
	registry := state.NewRegistry()
	registry.UpsertAccount(state.Account{
		Tag:               "work",
		AuthPath:          layout.AccountAuthPath("work"),
		Email:             "work@mail.com",
		AuthState:         state.AuthStateReady,
		LastStatusCheckAt: time.Now().Add(-5 * time.Minute).UTC().Format(time.RFC3339),
		LastKnownStatus: &state.StatusSnapshot{
			FiveHourUsedPct: &fiveHourUsed,
			WeeklyUsedPct:   &weeklyUsed,
			RawLimitSource:  "cached",
		},
	})
	require.NoError(t, state.SaveRegistry(layout.RegistryPath, registry))

	cmd, out := newTestCommandOutput()
	require.NoError(t, runStatus(cmd, nil))

	output := out.String()
	assert.Contains(t, output, "CHECKED")
	assert.Contains(t, output, "75%")
	assert.Contains(t, output, "50%")
	assert.Contains(t, output, "ready")
	assert.Contains(t, output, "ago")
}

func TestRunStatusDefaultShowsNeverForUncheckedAccount(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)
	failIfProbed(t)

	layout := paths.NewLayout(dir)
	registry := state.NewRegistry()
	registry.UpsertAccount(state.Account{
		Tag:      "work",
		AuthPath: layout.AccountAuthPath("work"),
	})
	require.NoError(t, state.SaveRegistry(layout.RegistryPath, registry))

	cmd, out := newTestCommandOutput()
	require.NoError(t, runStatus(cmd, nil))

	assert.Contains(t, out.String(), "never")
}

func TestRunStatusDefaultDoesNotRewriteRegistry(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)
	failIfProbed(t)

	layout := paths.NewLayout(dir)
	registry := state.NewRegistry()
	registry.UpsertAccount(state.Account{
		Tag:               "work",
		AuthPath:          layout.AccountAuthPath("work"),
		AuthState:         state.AuthStateReady,
		LastStatusCheckAt: "2026-06-12T00:00:00Z",
	})
	require.NoError(t, state.SaveRegistry(layout.RegistryPath, registry))

	cmd, _ := newTestCommandOutput()
	require.NoError(t, runStatus(cmd, nil))

	reloaded, err := state.LoadRegistry(layout.RegistryPath)
	require.NoError(t, err)
	account, ok := reloaded.FindAccount("work")
	require.True(t, ok)
	assert.Equal(t, "2026-06-12T00:00:00Z", account.LastStatusCheckAt)
}

func TestRunStatusRefreshPrintsAccounts(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)
	stubCheckCodexLoginStatus(t, true)
	stubProbeCodexAccountLimits(t, codex.UnknownRateLimitSnapshot("test"))

	layout := paths.NewLayout(dir)
	registry := state.NewRegistry()
	registry.UpsertAccount(state.Account{
		Tag:       "test",
		AuthPath:  "/tmp/new-auth.json",
		Email:     "test@mail.com",
		AuthState: state.AuthStateReady,
		CreatedAt: "2026-06-12T00:00:00Z",
		UpdatedAt: "2026-06-12T00:00:00Z",
	})
	require.NoError(t, state.SaveRegistry(layout.RegistryPath, registry))

	cmd, out := newTestCommandOutput()
	addRefreshFlag(cmd)

	require.NoError(t, runStatus(cmd, nil))
	assert.Contains(t, out.String(), "test")
	assert.Contains(t, out.String(), "test@mail.com")
	assert.Contains(t, out.String(), "ACTIVE")
	assert.Contains(t, out.String(), "*")
	assert.Contains(t, out.String(), "ready")
}

func TestRunStatusRefreshUsesUnknownForEmptyEmailAndAuthState(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)
	stubCheckCodexLoginStatus(t, false)

	layout := paths.NewLayout(dir)
	registry := state.NewRegistry()
	registry.UpsertAccount(state.Account{
		Tag: "test",
	})
	require.NoError(t, state.SaveRegistry(layout.RegistryPath, registry))

	cmd, out := newTestCommandOutput()
	addRefreshFlag(cmd)

	require.NoError(t, runStatus(cmd, nil))
	assert.Contains(t, out.String(), "needs_login")
	assert.Contains(t, out.String(), "unknown")
	assert.GreaterOrEqual(t, strings.Count(out.String(), "unknown"), 1)
}

func TestRunStatusRefreshesAuthStateNeedsLogin(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)
	stubCheckCodexLoginStatus(t, false)

	used := 25

	layout := paths.NewLayout(dir)
	registry := state.NewRegistry()
	registry.UpsertAccount(state.Account{
		Tag:       "work",
		AuthPath:  layout.AccountAuthPath("work"),
		AuthState: state.AuthStateReady,
		LastKnownStatus: &state.StatusSnapshot{
			FiveHourUsedPct: &used,
			RawLimitSource:  "old",
		},
	})
	require.NoError(t, state.SaveRegistry(layout.RegistryPath, registry))

	cmd, out := newTestCommandOutput()
	addRefreshFlag(cmd)

	require.NoError(t, runStatus(cmd, nil))
	assert.Contains(t, out.String(), "needs_login")

	registry, err := state.LoadRegistry(layout.RegistryPath)
	require.NoError(t, err)

	account, ok := registry.FindAccount("work")
	assert.True(t, ok)
	assert.Equal(t, state.AuthStateNeedsLogin, account.AuthState)
	assert.NotEmpty(t, account.LastStatusCheckAt)
	assert.Nil(t, account.LastKnownStatus)
}

func TestRunStatusRefreshPrintsQuota(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)
	stubCheckCodexLoginStatus(t, true)

	fiveHourUsed := 25
	weeklyUsed := 50
	stubProbeCodexAccountLimits(t, codex.RateLimitSnapshot{
		FiveHourUsedPct: &fiveHourUsed,
		WeeklyUsedPct:   &weeklyUsed,
		FiveHourResetIn: "1h",
		WeeklyResetIn:   "1d",
		RawLimitSource:  "test",
	})

	layout := paths.NewLayout(dir)
	registry := state.NewRegistry()
	registry.UpsertAccount(state.Account{
		Tag:      "work",
		AuthPath: layout.AccountAuthPath("work"),
		Email:    "work@mail.com",
	})
	require.NoError(t, state.SaveRegistry(layout.RegistryPath, registry))

	cmd, out := newTestCommandOutput()
	addRefreshFlag(cmd)

	require.NoError(t, runStatus(cmd, nil))

	assert.Contains(t, out.String(), "5H_LEFT")
	assert.Contains(t, out.String(), "WEEKLY_LEFT")
	assert.Contains(t, out.String(), "75%")
	assert.Contains(t, out.String(), "50%")
	assert.Contains(t, out.String(), "1h")
	assert.Contains(t, out.String(), "1d")

	registry, err := state.LoadRegistry(layout.RegistryPath)
	assert.NoError(t, err)

	account, ok := registry.FindAccount("work")
	assert.True(t, ok)

	assert.NotEmpty(t, account.LastStatusCheckAt)
	_, err = time.Parse(time.RFC3339, account.LastStatusCheckAt)
	assert.NoError(t, err)

	assert.NotNil(t, account.LastKnownStatus)
	assert.NotNil(t, account.LastKnownStatus.FiveHourUsedPct)
	assert.NotNil(t, account.LastKnownStatus.WeeklyUsedPct)

	assert.Equal(t, fiveHourUsed, *account.LastKnownStatus.FiveHourUsedPct)
	assert.Equal(t, weeklyUsed, *account.LastKnownStatus.WeeklyUsedPct)
	assert.Equal(t, "1h", account.LastKnownStatus.FiveHourResetIn)
	assert.Equal(t, "1d", account.LastKnownStatus.WeeklyResetIn)
	assert.Equal(t, "test", account.LastKnownStatus.RawLimitSource)
}

func TestRunStatusRefreshDoesNotProbeQuotaWhenNeedsLogin(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)
	stubCheckCodexLoginStatus(t, false)

	old := probeCodexAccountLimits
	t.Cleanup(func() {
		probeCodexAccountLimits = old
	})
	probeCodexAccountLimits = func(opts codex.ProbeAccountLimitsOptions) codex.RateLimitSnapshot {
		t.Fatalf("probeCodexAccountLimits should not be called")
		return codex.RateLimitSnapshot{}
	}

	layout := paths.NewLayout(dir)
	registry := state.NewRegistry()
	registry.UpsertAccount(state.Account{
		Tag:      "work",
		AuthPath: layout.AccountAuthPath("work"),
	})
	require.NoError(t, state.SaveRegistry(layout.RegistryPath, registry))

	cmd, _ := newTestCommandOutput()
	addRefreshFlag(cmd)

	require.NoError(t, runStatus(cmd, nil))
}

func TestRunStatusRefreshPrintsSingleAccountDetail(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)
	stubCheckCodexLoginStatus(t, true)

	fiveHourUsed := 25
	weeklyUsed := 50
	stubProbeCodexAccountLimits(t, codex.RateLimitSnapshot{
		FiveHourUsedPct: &fiveHourUsed,
		WeeklyUsedPct:   &weeklyUsed,
		FiveHourResetIn: "1h",
		WeeklyResetIn:   "1d",
		RawLimitSource:  "test-source",
	})

	layout := paths.NewLayout(dir)
	registry := state.NewRegistry()
	registry.UpsertAccount(state.Account{
		Tag:          "work",
		AuthPath:     layout.AccountAuthPath("work"),
		Email:        "work@mail.com",
		LastSwitchAt: "2026-06-12T01:00:00Z",
	})
	require.NoError(t, state.SaveRegistry(layout.RegistryPath, registry))

	cmd, out := newTestCommandOutput()
	cmd.Flags().StringP("tag", "t", "work", "")
	addRefreshFlag(cmd)

	require.NoError(t, runStatus(cmd, nil))

	registry, err := state.LoadRegistry(layout.RegistryPath)
	assert.NoError(t, err)

	account, ok := registry.FindAccount("work")
	assert.True(t, ok)
	assert.NotEmpty(t, account.LastStatusCheckAt)

	output := out.String()
	assert.Contains(t, output, "tag: work")
	assert.Contains(t, output, "active: yes")
	assert.Contains(t, output, "five_hour_left_pct: 75%")
	assert.Contains(t, output, "weekly_left_pct: 50%")
	assert.Contains(t, output, "five_hour_reset_in: 1h")
	assert.Contains(t, output, "weekly_reset_in: 1d")
	assert.Contains(t, output, "raw_limit_source: test-source")
	assert.Contains(t, output, "account: work@mail.com")
	assert.Contains(t, output, "auth_state: ready")
	assert.Contains(t, output, "auth_storage_path:")
	assert.Contains(t, output, "last_switch_at: 2026-06-12T01:00:00Z")
	assert.Contains(t, output, "last_status_check_at: "+account.LastStatusCheckAt)
}

func TestRunStatusSingleAccountDefaultShowsCache(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)
	failIfProbed(t)

	fiveHourUsed := 25

	layout := paths.NewLayout(dir)
	registry := state.NewRegistry()
	registry.UpsertAccount(state.Account{
		Tag:               "work",
		AuthPath:          layout.AccountAuthPath("work"),
		AuthState:         state.AuthStateReady,
		LastStatusCheckAt: "2026-06-12T00:00:00Z",
		LastKnownStatus: &state.StatusSnapshot{
			FiveHourUsedPct: &fiveHourUsed,
			RawLimitSource:  "cached",
		},
	})
	require.NoError(t, state.SaveRegistry(layout.RegistryPath, registry))

	cmd, out := newTestCommandOutput()
	cmd.Flags().StringP("tag", "t", "work", "")

	require.NoError(t, runStatus(cmd, nil))

	output := out.String()
	assert.Contains(t, output, "tag: work")
	assert.Contains(t, output, "five_hour_left_pct: 75%")
	assert.Contains(t, output, "auth_state: ready")
}

func TestRunStatusSingleAccountReturnsErrorForUnknownTag(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)

	layout := paths.NewLayout(dir)
	registry := state.NewRegistry()
	registry.UpsertAccount(state.Account{
		Tag:      "work",
		AuthPath: layout.AccountAuthPath("work"),
	})
	require.NoError(t, state.SaveRegistry(layout.RegistryPath, registry))

	cmd, _ := newTestCommandOutput()
	cmd.Flags().StringP("tag", "t", "personal", "")

	err := runStatus(cmd, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown account tag: personal")
}

func TestRunStatusRefreshSingleAccountNeedsLoginDetail(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)
	stubCheckCodexLoginStatus(t, false)

	old := probeCodexAccountLimits
	t.Cleanup(func() {
		probeCodexAccountLimits = old
	})
	probeCodexAccountLimits = func(opts codex.ProbeAccountLimitsOptions) codex.RateLimitSnapshot {
		t.Fatalf("probeCodexAccountLimits should not be called")
		return codex.RateLimitSnapshot{}
	}

	layout := paths.NewLayout(dir)
	registry := state.NewRegistry()
	registry.UpsertAccount(state.Account{
		Tag:      "work",
		AuthPath: layout.AccountAuthPath("work"),
	})
	require.NoError(t, state.SaveRegistry(layout.RegistryPath, registry))

	cmd, out := newTestCommandOutput()
	cmd.Flags().StringP("tag", "t", "work", "")
	addRefreshFlag(cmd)

	require.NoError(t, runStatus(cmd, nil))

	output := out.String()
	assert.Contains(t, output, "tag: work")
	assert.Contains(t, output, "auth_state: needs_login")
	assert.Contains(t, output, "five_hour_left_pct: unknown")
	assert.Contains(t, output, "weekly_left_pct: unknown")
}

func TestFormatCheckedAge(t *testing.T) {
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name string
		ts   string
		want string
	}{
		{"empty", "", "never"},
		{"malformed", "not-a-time", "unknown"},
		{"just now", now.Add(-10 * time.Second).Format(time.RFC3339), "just now"},
		{"minutes", now.Add(-5 * time.Minute).Format(time.RFC3339), "5m ago"},
		{"hours", now.Add(-3 * time.Hour).Format(time.RFC3339), "3h ago"},
		{"days", now.Add(-50 * time.Hour).Format(time.RFC3339), "2d ago"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, formatCheckedAge(c.ts, now))
		})
	}
}
