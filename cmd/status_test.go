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

func TestRunStatusPrintsNoAccounts(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)

	cmd, out := newTestCommandOutput()

	err := runStatus(cmd, nil)
	assert.NoError(t, err)
	assert.Equal(t, "no accounts\n", out.String())
}

func TestRunStatusPrintsAccounts(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)
	stubCheckCodexLoginStatus(t, true)
	stubProbeCodexAccountLimits(t, codex.UnknownRateLimitSnapshot("test"))

	layout := paths.NewLayout(dir)
	registry := state.NewRegistry()
	acc := state.Account{
		Tag:       "test",
		AuthPath:  "/tmp/new-auth.json",
		Email:     "test@mail.com",
		AuthState: state.AuthStateReady,
		CreatedAt: "2026-06-12T00:00:00Z",
		UpdatedAt: "2026-06-12T00:00:00Z",
	}
	registry.UpsertAccount(acc)
	err := state.SaveRegistry(layout.RegistryPath, registry)
	assert.NoError(t, err)

	cmd, out := newTestCommandOutput()

	err = runStatus(cmd, nil)
	assert.NoError(t, err)
	assert.Contains(t, out.String(), "test")
	assert.Contains(t, out.String(), "test@mail.com")
	assert.Contains(t, out.String(), "ACTIVE")
	assert.Contains(t, out.String(), "*")
	assert.Contains(t, out.String(), "ready")
}

func TestRunStatusUsesUnknownForEmptyEmailAndAuthState(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)
	stubCheckCodexLoginStatus(t, false)

	layout := paths.NewLayout(dir)
	registry := state.NewRegistry()
	acc := state.Account{
		Tag: "test",
	}
	registry.UpsertAccount(acc)
	err := state.SaveRegistry(layout.RegistryPath, registry)
	assert.NoError(t, err)

	cmd, out := newTestCommandOutput()

	err = runStatus(cmd, nil)
	assert.NoError(t, err)
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
	err := state.SaveRegistry(layout.RegistryPath, registry)
	require.NoError(t, err)

	cmd, out := newTestCommandOutput()

	err = runStatus(cmd, nil)
	require.NoError(t, err)
	assert.Contains(t, out.String(), "needs_login")

	registry, err = state.LoadRegistry(layout.RegistryPath)
	require.NoError(t, err)

	account, ok := registry.FindAccount("work")
	assert.True(t, ok)
	assert.Equal(t, state.AuthStateNeedsLogin, account.AuthState)
	assert.NotEmpty(t, account.LastStatusCheckAt)
	assert.Nil(t, account.LastKnownStatus)
}

func TestRunStatusPrintsQuota(t *testing.T) {
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
	err := state.SaveRegistry(layout.RegistryPath, registry)
	require.NoError(t, err)

	cmd, out := newTestCommandOutput()

	err = runStatus(cmd, nil)
	require.NoError(t, err)

	assert.Contains(t, out.String(), "5H_LEFT")
	assert.Contains(t, out.String(), "WEEKLY_LEFT")
	assert.Contains(t, out.String(), "75%")
	assert.Contains(t, out.String(), "50%")
	assert.Contains(t, out.String(), "1h")
	assert.Contains(t, out.String(), "1d")

	registry, err = state.LoadRegistry(layout.RegistryPath)
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

func TestRunStatusDoesNotProbeQuotaWhenNeedsLogin(t *testing.T) {
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

	err := runStatus(cmd, nil)
	require.NoError(t, err)
}

func TestRunStatusPrintsSingleAccountDetail(t *testing.T) {
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
	assert.NoError(t, state.SaveRegistry(layout.RegistryPath, registry))

	cmd, out := newTestCommandOutput()
	cmd.Flags().StringP("tag", "t", "work", "")

	err := runStatus(cmd, nil)
	assert.NoError(t, err)

	registry, err = state.LoadRegistry(layout.RegistryPath)
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

func TestRunStatusSingleAccountReturnsErrorForUnknownTag(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvHome, dir)

	layout := paths.NewLayout(dir)
	registry := state.NewRegistry()
	registry.UpsertAccount(state.Account{
		Tag:      "work",
		AuthPath: layout.AccountAuthPath("work"),
	})
	assert.NoError(t, state.SaveRegistry(layout.RegistryPath, registry))

	cmd, _ := newTestCommandOutput()
	cmd.Flags().StringP("tag", "t", "personal", "")

	err := runStatus(cmd, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown account tag: personal")
}

func TestRunStatusSingleAccountNeedsLoginDetail(t *testing.T) {
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
	assert.NoError(t, state.SaveRegistry(layout.RegistryPath, registry))

	cmd, out := newTestCommandOutput()
	cmd.Flags().StringP("tag", "t", "work", "")

	err := runStatus(cmd, nil)
	assert.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "tag: work")
	assert.Contains(t, output, "auth_state: needs_login")
	assert.Contains(t, output, "five_hour_left_pct: unknown")
	assert.Contains(t, output, "weekly_left_pct: unknown")
}
