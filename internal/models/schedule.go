package models

import "time"

// Schedule alert statuses.
const (
	ScheduleOK    = "ok"
	ScheduleAlert = "alert"
	ScheduleError = "error"
)

// Schedule runs a target URL (or a token's action chain) on a cron interval and
// alerts via a notify URL when a monitoring rule trips.
type Schedule struct {
	UUID      string `json:"uuid"`
	TokenID   string `json:"token_id,omitempty"`
	Name      string `json:"name"`
	Cron      string `json:"cron"`
	TargetURL string `json:"target_url"`
	Method    string `json:"method"`
	Body      string `json:"body"`

	RunActions bool `json:"run_actions"` // run the token's action chain instead of an HTTP request

	// Monitoring / alert rules.
	ExpectStatus int    `json:"expect_status"` // 0 = any 2xx is OK
	Keyword      string `json:"keyword"`       // alert if the response body lacks this
	CheckSSL     bool   `json:"check_ssl"`     // alert if the TLS cert expires soon
	SSLDays      int    `json:"ssl_days"`      // threshold in days

	NotifyURL string `json:"notify_url,omitempty"` // shoutrrr URL; never returned in plaintext by the API
	Enabled   bool   `json:"enabled"`

	LastRun     *time.Time `json:"last_run,omitempty"`
	NextRun     *time.Time `json:"next_run,omitempty"`
	LastStatus  string     `json:"last_status"`
	LastMessage string     `json:"last_message"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ScheduleRun is one execution record of a schedule.
type ScheduleRun struct {
	ID         string    `json:"id"`
	ScheduleID string    `json:"schedule_id"`
	Status     string    `json:"status"`
	StatusCode int       `json:"status_code"`
	Message    string    `json:"message"`
	DurationMS int       `json:"duration_ms"`
	CreatedAt  time.Time `json:"created_at"`
}
