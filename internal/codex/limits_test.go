package codex

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatResetInMinutes(t *testing.T) {
	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	resetsAt := now.Add(14 * time.Minute).Unix()
	assert.Equal(t, "14m", FormatResetIn(resetsAt, now))
}

func TestFormatResetInHoursAndMinutes(t *testing.T) {
	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	resetsAt := now.Add(1*time.Hour + 30*time.Minute).Unix()
	assert.Equal(t, "1h 30m", FormatResetIn(resetsAt, now))
}

func TestFormatResetInDaysAndHours(t *testing.T) {
	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	resetsAt := now.Add(2*24*time.Hour + 5*time.Hour).Unix()
	assert.Equal(t, "2d 5h", FormatResetIn(resetsAt, now))
}

func TestFormatResetInPastTime(t *testing.T) {
	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	resetsAt := now.Add(-10 * time.Minute).Unix()
	assert.Equal(t, "0m", FormatResetIn(resetsAt, now))
}

func TestUnknownRateLimitSnapshot(t *testing.T) {
	snapshot := UnknownRateLimitSnapshot("app-server failed")
	assert.Nil(t, snapshot.FiveHourUsedPct)
	assert.Nil(t, snapshot.WeeklyUsedPct)
	assert.Equal(t, "app-server failed", snapshot.RawLimitSource)
}

func TestNormalizeAppServerRateLimits(t *testing.T) {
	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	fiveHourUsed := 68
	weeklyUsed := 41
	fiveHourReset := now.Add(2*time.Hour + 30*time.Minute).Unix()
	weeklyReset := now.Add(6*24*time.Hour + 4*time.Hour).Unix()

	snapshot := normalizeAppServerRateLimits(appServerRateLimitResponse{
		RateLimits: appServerRateLimits{
			Primary: &appServerRateLimitWindow{
				UsedPercent: &fiveHourUsed,
				ResetsAt:    &fiveHourReset,
			},
			Secondary: &appServerRateLimitWindow{
				UsedPercent: &weeklyUsed,
				ResetsAt:    &weeklyReset,
			},
			PlanType: "plus",
		},
	}, now)

	assert.NotNil(t, snapshot.FiveHourUsedPct)
	assert.Equal(t, 68, *snapshot.FiveHourUsedPct)
	assert.NotNil(t, snapshot.WeeklyUsedPct)
	assert.Equal(t, 41, *snapshot.WeeklyUsedPct)
	assert.Equal(t, "2h 30m", snapshot.FiveHourResetIn)
	assert.Equal(t, "6d 4h", snapshot.WeeklyResetIn)
	assert.Equal(t, "plus", snapshot.PlanType)
}

func TestNormalizeAppServerRateLimitsAllowsMissingWindows(t *testing.T) {
	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	snapshot := normalizeAppServerRateLimits(appServerRateLimitResponse{
		RateLimits: appServerRateLimits{},
	}, now)

	assert.Nil(t, snapshot.FiveHourUsedPct)
	assert.Nil(t, snapshot.WeeklyUsedPct)
	assert.Empty(t, snapshot.FiveHourResetIn)
	assert.Empty(t, snapshot.WeeklyResetIn)
	assert.Equal(t, "app-server account/rateLimits/read", snapshot.RawLimitSource)
}

func TestProbeAccountLimitUsesAppServerRequester(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "codex-home")
	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	fiveHourUsed := 25
	weeklyUser := 50
	fiveHourReset := now.Add(time.Hour).Unix()
	weeklyReset := now.Add(24 * time.Hour).Unix()

	snapshot := ProbeAccountLimits(ProbeAccountLimitsOptions{
		CodexHome: dir,
		Now:       now,
		Requester: func(opts AppServerRequestOptions, out any) error {
			assert.Equal(t, dir, opts.CodexHome)
			assert.Equal(t, "account/rateLimits/read", opts.Method)

			response := out.(*appServerRateLimitResponse)
			*response = appServerRateLimitResponse{
				RateLimits: appServerRateLimits{
					Primary: &appServerRateLimitWindow{
						UsedPercent: &fiveHourUsed,
						ResetsAt:    &fiveHourReset,
					},
					Secondary: &appServerRateLimitWindow{
						UsedPercent: &weeklyUser,
						ResetsAt:    &weeklyReset,
					},
					PlanType: "plus",
				},
			}
			return nil
		},
	})

	assert.NotNil(t, snapshot.FiveHourUsedPct)
	assert.Equal(t, 25, *snapshot.FiveHourUsedPct)
	assert.NotNil(t, snapshot.WeeklyUsedPct)
	assert.Equal(t, 50, *snapshot.WeeklyUsedPct)
	assert.Equal(t, "1h", snapshot.FiveHourResetIn)
	assert.Equal(t, "1d", snapshot.WeeklyResetIn)
	assert.Equal(t, "plus", snapshot.PlanType)
}

func TestProbeAccountLimitReturnsUnknownWhenRequesterFails(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "codex-home")
	snapshot := ProbeAccountLimits(ProbeAccountLimitsOptions{
		CodexHome: dir,
		Requester: func(opts AppServerRequestOptions, out any) error {
			return assert.AnError
		},
	})

	assert.Nil(t, snapshot.FiveHourUsedPct)
	assert.Nil(t, snapshot.WeeklyUsedPct)
	assert.Contains(t, snapshot.RawLimitSource, assert.AnError.Error())
}
