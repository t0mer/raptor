-- Phase 4: Custom Actions engine — action definitions and per-request run logs. (PostgreSQL)

CREATE TABLE actions (
    uuid       TEXT PRIMARY KEY,
    token_id   TEXT NOT NULL REFERENCES tokens (uuid) ON DELETE CASCADE,
    type       TEXT NOT NULL,
    position   INTEGER NOT NULL DEFAULT 0,
    name       TEXT NOT NULL DEFAULT '',
    disabled   INTEGER NOT NULL DEFAULT 0,
    parameters TEXT NOT NULL DEFAULT '{}',
    queue      INTEGER NOT NULL DEFAULT 0,
    delay      INTEGER NOT NULL DEFAULT 0,
    condition  TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX idx_actions_token ON actions (token_id, position);

CREATE TABLE action_runs (
    id          TEXT PRIMARY KEY,
    request_id  TEXT NOT NULL REFERENCES requests (uuid) ON DELETE CASCADE,
    action_id   TEXT NOT NULL,
    action_type TEXT NOT NULL,
    action_name TEXT NOT NULL DEFAULT '',
    position    INTEGER NOT NULL DEFAULT 0,
    output      TEXT NOT NULL DEFAULT '',
    error       TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL
);

CREATE INDEX idx_action_runs_request ON action_runs (request_id, position);
