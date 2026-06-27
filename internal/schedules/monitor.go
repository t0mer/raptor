// Package schedules runs cron-scheduled checks against target URLs (or a token's
// action chain), records run history, and alerts via a notify URL when a
// monitoring rule trips (status / keyword / uptime / SSL expiry).
package schedules

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"

	"github.com/t0mer/raptor/internal/actions"
	"github.com/t0mer/raptor/internal/models"
	"github.com/t0mer/raptor/internal/notify"
	"github.com/t0mer/raptor/internal/store"
)

// maxProbeBytes caps how much of a monitored response is read for keyword checks.
// The body itself is never stored or returned, only the boolean match result.
const maxProbeBytes = 256 << 10 // 256 KiB

// cronParser accepts standard 5-field cron expressions.
var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// Runner ticks periodically and executes any schedule that has come due.
type Runner struct {
	store    *store.Store
	actions  *actions.Service
	client   *http.Client
	logger   *slog.Logger
	interval time.Duration
	now      func() time.Time
}

// Option configures a Runner.
type Option func(*Runner)

// WithLogger sets the logger.
func WithLogger(l *slog.Logger) Option {
	return func(r *Runner) {
		if l != nil {
			r.logger = l
		}
	}
}

// WithInterval overrides the tick interval (default 30s).
func WithInterval(d time.Duration) Option {
	return func(r *Runner) {
		if d > 0 {
			r.interval = d
		}
	}
}

// New builds a Runner.
func New(st *store.Store, actionsSvc *actions.Service, opts ...Option) *Runner {
	r := &Runner{
		store:    st,
		actions:  actionsSvc,
		client:   &http.Client{Timeout: 20 * time.Second},
		logger:   slog.Default(),
		interval: 30 * time.Second,
		now:      time.Now,
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Start runs the tick loop until ctx is cancelled.
func (r *Runner) Start(ctx context.Context) {
	r.logger.Info("schedule runner started", "interval", r.interval)
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.tick(ctx)
		}
	}
}

func (r *Runner) tick(ctx context.Context) {
	schedules, err := r.store.ListEnabledSchedules(ctx)
	if err != nil {
		r.logger.Error("list schedules", "error", err)
		return
	}
	now := r.now().UTC()
	for _, sc := range schedules {
		if sc.NextRun != nil && now.Before(*sc.NextRun) {
			continue // not due yet
		}
		r.Run(ctx, sc)
	}
}

// Run executes a single schedule: performs its check, records the run, computes
// the next run time, and alerts on a non-ok result.
func (r *Runner) Run(ctx context.Context, sc *models.Schedule) *models.ScheduleRun {
	start := r.now()
	status, code, message := r.check(ctx, sc)
	dur := int(time.Since(start).Milliseconds())

	run := &models.ScheduleRun{
		ID:         uuid.NewString(),
		ScheduleID: sc.UUID,
		Status:     status,
		StatusCode: code,
		Message:    message,
		DurationMS: dur,
	}
	if err := r.store.CreateScheduleRun(ctx, run); err != nil {
		r.logger.Error("record schedule run", "schedule", sc.UUID, "error", err)
	}

	next := r.nextRun(sc, start.UTC())
	if err := r.store.RecordScheduleResult(ctx, sc.UUID, start.UTC(), next, status, message); err != nil {
		r.logger.Error("record schedule result", "schedule", sc.UUID, "error", err)
	}

	if status != models.ScheduleOK {
		title := fmt.Sprintf("Raptor alert: %s", scheduleName(sc))
		if err := notify.Send(sc.NotifyURL, title, message); err != nil {
			r.logger.Warn("send alert", "schedule", sc.UUID, "error", err)
		}
	}
	return run
}

// nextRun computes the next fire time from the cron expression; on a parse error
// it falls back to one hour out so a bad expression doesn't hot-loop.
func (r *Runner) nextRun(sc *models.Schedule, from time.Time) time.Time {
	sched, err := cronParser.Parse(sc.Cron)
	if err != nil {
		return from.Add(time.Hour)
	}
	return sched.Next(from)
}

// check performs the schedule's monitoring check and returns (status, httpCode, message).
func (r *Runner) check(ctx context.Context, sc *models.Schedule) (string, int, string) {
	if sc.RunActions {
		return r.runActions(ctx, sc)
	}
	return r.httpCheck(ctx, sc)
}

func (r *Runner) httpCheck(ctx context.Context, sc *models.Schedule) (string, int, string) {
	if strings.TrimSpace(sc.TargetURL) == "" {
		return models.ScheduleError, 0, "no target URL configured"
	}
	method := sc.Method
	if method == "" {
		method = http.MethodGet
	}
	var bodyReader io.Reader
	if sc.Body != "" {
		bodyReader = strings.NewReader(sc.Body)
	}
	req, err := http.NewRequestWithContext(ctx, method, sc.TargetURL, bodyReader)
	if err != nil {
		return models.ScheduleError, 0, "invalid request: " + err.Error()
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return models.ScheduleAlert, 0, "host unreachable: " + err.Error()
	}
	defer resp.Body.Close()

	// SSL expiry check.
	if sc.CheckSSL && resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		if msg, alert := sslExpiry(resp.TLS, sc.SSLDays, r.now()); alert {
			return models.ScheduleAlert, resp.StatusCode, msg
		}
	}

	// Status check.
	if sc.ExpectStatus > 0 {
		if resp.StatusCode != sc.ExpectStatus {
			return models.ScheduleAlert, resp.StatusCode, fmt.Sprintf("status %d, expected %d", resp.StatusCode, sc.ExpectStatus)
		}
	} else if resp.StatusCode >= 400 {
		return models.ScheduleAlert, resp.StatusCode, fmt.Sprintf("status %d", resp.StatusCode)
	}

	// Keyword check (body content is never stored — only the match result).
	if sc.Keyword != "" {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxProbeBytes))
		if !strings.Contains(string(body), sc.Keyword) {
			return models.ScheduleAlert, resp.StatusCode, fmt.Sprintf("keyword %q not found", sc.Keyword)
		}
	}

	return models.ScheduleOK, resp.StatusCode, fmt.Sprintf("ok (status %d)", resp.StatusCode)
}

func (r *Runner) runActions(ctx context.Context, sc *models.Schedule) (string, int, string) {
	if r.actions == nil || sc.TokenID == "" {
		return models.ScheduleError, 0, "run_actions requires a token and the actions engine"
	}
	tok, err := r.store.GetToken(ctx, sc.TokenID)
	if err != nil {
		return models.ScheduleError, 0, "token not found"
	}
	req := &models.Request{TokenID: tok.UUID, Type: models.RequestTypeWeb, Method: http.MethodGet}
	_, results, err := r.actions.Execute(ctx, req, tok)
	if err != nil {
		return models.ScheduleError, 0, "execute actions: " + err.Error()
	}
	for _, res := range results {
		if res.Err != nil {
			return models.ScheduleAlert, 0, "action error: " + res.Err.Error()
		}
	}
	return models.ScheduleOK, 0, fmt.Sprintf("ran %d actions", len(results))
}

// sslExpiry reports whether the leaf certificate expires within `days`.
func sslExpiry(state *tls.ConnectionState, days int, now time.Time) (string, bool) {
	if days <= 0 {
		days = 14
	}
	leaf := state.PeerCertificates[0]
	remaining := leaf.NotAfter.Sub(now)
	if remaining < time.Duration(days)*24*time.Hour {
		return fmt.Sprintf("TLS certificate expires in %d days (%s)", int(remaining.Hours()/24), leaf.NotAfter.Format("2006-01-02")), true
	}
	return "", false
}

func scheduleName(sc *models.Schedule) string {
	if sc.Name != "" {
		return sc.Name
	}
	if sc.TargetURL != "" {
		return sc.TargetURL
	}
	return sc.UUID
}
