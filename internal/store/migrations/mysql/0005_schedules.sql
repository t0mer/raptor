-- Phase 5: cron schedules with monitoring/alerting and a per-run history. (MySQL)

CREATE TABLE schedules (
    uuid          VARCHAR(64) PRIMARY KEY,
    token_id      VARCHAR(64) NOT NULL DEFAULT '',
    name          VARCHAR(255) NOT NULL DEFAULT '',
    cron          VARCHAR(255) NOT NULL,
    target_url    VARCHAR(2048) NOT NULL DEFAULT '',
    method        VARCHAR(32) NOT NULL DEFAULT 'GET',
    body          LONGTEXT NOT NULL DEFAULT (''),
    run_actions   TINYINT NOT NULL DEFAULT 0,
    expect_status INT NOT NULL DEFAULT 0,
    keyword       VARCHAR(512) NOT NULL DEFAULT '',
    check_ssl     TINYINT NOT NULL DEFAULT 0,
    ssl_days      INT NOT NULL DEFAULT 14,
    notify_url    VARCHAR(2048) NOT NULL DEFAULT '',
    enabled       TINYINT NOT NULL DEFAULT 1,
    last_run      VARCHAR(64),
    next_run      VARCHAR(64),
    last_status   VARCHAR(64) NOT NULL DEFAULT '',
    last_message  VARCHAR(1024) NOT NULL DEFAULT '',
    created_at    VARCHAR(64) NOT NULL,
    updated_at    VARCHAR(64) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE schedule_runs (
    id          VARCHAR(64) PRIMARY KEY,
    schedule_id VARCHAR(64) NOT NULL,
    status      VARCHAR(64) NOT NULL DEFAULT '',
    status_code INT NOT NULL DEFAULT 0,
    message     VARCHAR(1024) NOT NULL DEFAULT '',
    duration_ms BIGINT NOT NULL DEFAULT 0,
    created_at  VARCHAR(64) NOT NULL,
    CONSTRAINT fk_schedule_runs FOREIGN KEY (schedule_id) REFERENCES schedules (uuid) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_schedule_runs ON schedule_runs (schedule_id, created_at DESC);
