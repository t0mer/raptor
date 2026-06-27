package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/t0mer/raptor/internal/models"
)

const scheduleColumns = `uuid, token_id, name, cron, target_url, method, body, run_actions,
	expect_status, keyword, check_ssl, ssl_days, notify_url, enabled,
	last_run, next_run, last_status, last_message, created_at, updated_at`

// CreateSchedule inserts a schedule, encrypting the notify URL at rest.
func (s *Store) CreateSchedule(ctx context.Context, sc *models.Schedule) error {
	now := time.Now().UTC()
	if sc.CreatedAt.IsZero() {
		sc.CreatedAt = now
	}
	sc.UpdatedAt = now
	notify, err := s.encryptSecret(sc.NotifyURL)
	if err != nil {
		return fmt.Errorf("encrypt notify url: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO schedules (`+scheduleColumns+`)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		sc.UUID, sc.TokenID, sc.Name, sc.Cron, sc.TargetURL, sc.Method, sc.Body, boolToInt(sc.RunActions),
		sc.ExpectStatus, sc.Keyword, boolToInt(sc.CheckSSL), sc.SSLDays, notify, boolToInt(sc.Enabled),
		nullTime(sc.LastRun), nullTime(sc.NextRun), sc.LastStatus, sc.LastMessage,
		nowRFC3339(sc.CreatedAt), nowRFC3339(sc.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("insert schedule: %w", err)
	}
	return nil
}

// GetSchedule returns a schedule by id (notify URL decrypted).
func (s *Store) GetSchedule(ctx context.Context, uuid string) (*models.Schedule, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+scheduleColumns+` FROM schedules WHERE uuid = ?`, uuid)
	return s.scanSchedule(row)
}

// ListSchedules returns all schedules, newest first.
func (s *Store) ListSchedules(ctx context.Context) ([]*models.Schedule, error) {
	return s.querySchedules(ctx, `SELECT `+scheduleColumns+` FROM schedules ORDER BY created_at DESC`)
}

// ListEnabledSchedules returns enabled schedules (for the runner).
func (s *Store) ListEnabledSchedules(ctx context.Context) ([]*models.Schedule, error) {
	return s.querySchedules(ctx, `SELECT `+scheduleColumns+` FROM schedules WHERE enabled = 1`)
}

func (s *Store) querySchedules(ctx context.Context, query string, args ...any) ([]*models.Schedule, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query schedules: %w", err)
	}
	defer rows.Close()
	var out []*models.Schedule
	for rows.Next() {
		sc, err := s.scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sc)
	}
	return out, rows.Err()
}

// UpdateSchedule persists a schedule's mutable fields (re-encrypting notify URL).
func (s *Store) UpdateSchedule(ctx context.Context, sc *models.Schedule) error {
	sc.UpdatedAt = time.Now().UTC()
	notify, err := s.encryptSecret(sc.NotifyURL)
	if err != nil {
		return fmt.Errorf("encrypt notify url: %w", err)
	}
	res, err := s.db.ExecContext(ctx, `UPDATE schedules SET
		token_id=?, name=?, cron=?, target_url=?, method=?, body=?, run_actions=?,
		expect_status=?, keyword=?, check_ssl=?, ssl_days=?, notify_url=?, enabled=?, updated_at=?
		WHERE uuid=?`,
		sc.TokenID, sc.Name, sc.Cron, sc.TargetURL, sc.Method, sc.Body, boolToInt(sc.RunActions),
		sc.ExpectStatus, sc.Keyword, boolToInt(sc.CheckSSL), sc.SSLDays, notify, boolToInt(sc.Enabled),
		nowRFC3339(sc.UpdatedAt), sc.UUID,
	)
	if err != nil {
		return fmt.Errorf("update schedule: %w", err)
	}
	return requireAffected(res)
}

// DeleteSchedule removes a schedule and its run history.
func (s *Store) DeleteSchedule(ctx context.Context, uuid string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM schedules WHERE uuid = ?`, uuid)
	if err != nil {
		return fmt.Errorf("delete schedule: %w", err)
	}
	return requireAffected(res)
}

// RecordScheduleResult updates a schedule's last/next run and status.
func (s *Store) RecordScheduleResult(ctx context.Context, uuid string, lastRun, nextRun time.Time, status, message string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE schedules SET
		last_run=?, next_run=?, last_status=?, last_message=? WHERE uuid=?`,
		nowRFC3339(lastRun), nowRFC3339(nextRun), status, message, uuid)
	return err
}

// CreateScheduleRun appends a run-history record.
func (s *Store) CreateScheduleRun(ctx context.Context, r *models.ScheduleRun) error {
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO schedule_runs (id, schedule_id, status, status_code, message, duration_ms, created_at)
		 VALUES (?,?,?,?,?,?,?)`,
		r.ID, r.ScheduleID, r.Status, r.StatusCode, r.Message, r.DurationMS, nowRFC3339(r.CreatedAt))
	return err
}

// ListScheduleRuns returns recent run history for a schedule.
func (s *Store) ListScheduleRuns(ctx context.Context, scheduleID string, limit int) ([]*models.ScheduleRun, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, schedule_id, status, status_code, message, duration_ms, created_at
		 FROM schedule_runs WHERE schedule_id = ? ORDER BY created_at DESC LIMIT ?`, scheduleID, limit)
	if err != nil {
		return nil, fmt.Errorf("query schedule_runs: %w", err)
	}
	defer rows.Close()
	var out []*models.ScheduleRun
	for rows.Next() {
		var (
			r       models.ScheduleRun
			created string
		)
		if err := rows.Scan(&r.ID, &r.ScheduleID, &r.Status, &r.StatusCode, &r.Message, &r.DurationMS, &created); err != nil {
			return nil, err
		}
		r.CreatedAt, _ = parseTime(created)
		out = append(out, &r)
	}
	return out, rows.Err()
}

func (s *Store) scanSchedule(sc scanner) (*models.Schedule, error) {
	var (
		m                             models.Schedule
		runActions, checkSSL, enabled int
		notify                        string
		lastRun, nextRun              sql.NullString
		created, updated              string
	)
	err := sc.Scan(
		&m.UUID, &m.TokenID, &m.Name, &m.Cron, &m.TargetURL, &m.Method, &m.Body, &runActions,
		&m.ExpectStatus, &m.Keyword, &checkSSL, &m.SSLDays, &notify, &enabled,
		&lastRun, &nextRun, &m.LastStatus, &m.LastMessage, &created, &updated,
	)
	if err != nil {
		return nil, mapNoRows(err)
	}
	m.RunActions = runActions != 0
	m.CheckSSL = checkSSL != 0
	m.Enabled = enabled != 0
	if m.NotifyURL, err = s.decryptSecret(notify); err != nil {
		return nil, fmt.Errorf("decrypt notify url: %w", err)
	}
	if lastRun.Valid && lastRun.String != "" {
		if t, err := parseTime(lastRun.String); err == nil {
			m.LastRun = &t
		}
	}
	if nextRun.Valid && nextRun.String != "" {
		if t, err := parseTime(nextRun.String); err == nil {
			m.NextRun = &t
		}
	}
	m.CreatedAt, _ = parseTime(created)
	m.UpdatedAt, _ = parseTime(updated)
	return &m, nil
}
