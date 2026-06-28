-- Phase 5: cron schedules with monitoring/alerting and a per-run history. (PostgreSQL)

CREATE TABLE schedules (
    uuid          TEXT PRIMARY KEY,
    token_id      TEXT NOT NULL DEFAULT '',
    name          TEXT NOT NULL DEFAULT '',
    cron          TEXT NOT NULL,
    target_url    TEXT NOT NULL DEFAULT '',
    method        TEXT NOT NULL DEFAULT 'GET',
    body          TEXT NOT NULL DEFAULT '',
    run_actions   INTEGER NOT NULL DEFAULT 0,
    expect_status INTEGER NOT NULL DEFAULT 0,
    keyword       TEXT NOT NULL DEFAULT '',
    check_ssl     INTEGER NOT NULL DEFAULT 0,
    ssl_days      INTEGER NOT NULL DEFAULT 14,
    notify_url    TEXT NOT NULL DEFAULT '',
    enabled       INTEGER NOT NULL DEFAULT 1,
    last_run      TEXT,
    next_run      TEXT,
    last_status   TEXT NOT NULL DEFAULT '',
    last_message  TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL
);

CREATE TABLE schedule_runs (
    id          TEXT PRIMARY KEY,
    schedule_id TEXT NOT NULL REFERENCES schedules (uuid) ON DELETE CASCADE,
    status      TEXT NOT NULL DEFAULT '',
    status_code INTEGER NOT NULL DEFAULT 0,
    message     TEXT NOT NULL DEFAULT '',
    duration_ms BIGINT NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL
);

CREATE INDEX idx_schedule_runs ON schedule_runs (schedule_id, created_at DESC);
