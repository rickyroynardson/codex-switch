package codex

import (
	"fmt"
	"time"
)

type RateLimitSnapshot struct {
	FiveHourUsedPct *int
	WeeklyUsedPct   *int
	FiveHourResetIn string
	WeeklyResetIn   string
	RawLimitSource  string
	PlanType        string
}

type appServerRateLimitWindow struct {
	UsedPercent        *int   `json:"usedPercent"`
	WindowDurationMins *int   `json:"windowDurationMins"`
	ResetsAt           *int64 `json:"resetsAt"`
}

type appServerRateLimits struct {
	Primary   *appServerRateLimitWindow `json:"primary"`
	Secondary *appServerRateLimitWindow `json:"secondary"`
	PlanType  string
}

type appServerRateLimitResponse struct {
	RateLimits appServerRateLimits `json:"rateLimits"`
}

type ProbeAccountLimitsOptions struct {
	CodexHome    string
	CodexCommand string
	Requester    AppServerRequester
	Now          time.Time
}

func UnknownRateLimitSnapshot(source string) RateLimitSnapshot {
	if source == "" {
		source = "unknown"
	}

	return RateLimitSnapshot{
		RawLimitSource: source,
	}
}

func FormatResetIn(resetAtUnixSeconds int64, now time.Time) string {
	deltaSeconds := resetAtUnixSeconds - now.Unix()
	if deltaSeconds < 0 {
		deltaSeconds = 0
	}

	totalMinutes := deltaSeconds / 60
	if totalMinutes < 60 {
		return fmt.Sprintf("%dm", totalMinutes)
	}

	hours := totalMinutes / 60
	minutes := totalMinutes % 60
	days := hours / 24
	remainingHours := hours % 24

	if days > 0 {
		if remainingHours > 0 {
			return fmt.Sprintf("%dd %dh", days, remainingHours)
		}
		if minutes > 0 {
			return fmt.Sprintf("%dd %dm", days, minutes)
		}
		return fmt.Sprintf("%dd", days)
	}

	if minutes == 0 {
		return fmt.Sprintf("%dh", hours)
	}

	return fmt.Sprintf("%dh %dm", hours, minutes)
}

func normalizeAppServerRateLimits(response appServerRateLimitResponse, now time.Time) RateLimitSnapshot {
	snapshot := RateLimitSnapshot{
		RawLimitSource: "app-server account/rateLimits/read",
		PlanType:       response.RateLimits.PlanType,
	}

	if primary := response.RateLimits.Primary; primary != nil {
		snapshot.FiveHourUsedPct = primary.UsedPercent
		if primary.ResetsAt != nil {
			snapshot.FiveHourResetIn = FormatResetIn(*primary.ResetsAt, now)
		}
	}

	if secondary := response.RateLimits.Secondary; secondary != nil {
		snapshot.WeeklyUsedPct = secondary.UsedPercent
		if secondary.ResetsAt != nil {
			snapshot.WeeklyResetIn = FormatResetIn(*secondary.ResetsAt, now)
		}
	}

	return snapshot
}

func ProbeAccountLimits(opts ProbeAccountLimitsOptions) RateLimitSnapshot {
	requester := opts.Requester
	if requester == nil {
		requester = RequestAppServer
	}

	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}

	var response appServerRateLimitResponse
	err := requester(AppServerRequestOptions{
		CodexHome:    opts.CodexHome,
		CodexCommand: opts.CodexCommand,
		Method:       "account/rateLimits/read",
		Params:       map[string]any{},
	}, &response)
	if err != nil {
		return UnknownRateLimitSnapshot(err.Error())
	}

	return normalizeAppServerRateLimits(response, now)
}
