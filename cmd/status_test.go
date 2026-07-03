package cmd

import (
	"bytes"
	"strings"
	"testing"

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

	layout := paths.NewLayout(dir)
	registry := state.NewRegistry()
	registry.UpsertAccount(state.Account{
		Tag:       "work",
		AuthPath:  layout.AccountAuthPath("work"),
		AuthState: state.AuthStateReady,
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
