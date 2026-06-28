-- Phase 1 core schema: tokens, requests, files. (PostgreSQL)

CREATE TABLE tokens (
    uuid                 TEXT PRIMARY KEY,
    alias                TEXT NOT NULL DEFAULT '',
    default_status       INTEGER NOT NULL DEFAULT 200,
    default_content      TEXT NOT NULL DEFAULT '',
    default_content_type TEXT NOT NULL DEFAULT 'text/plain',
    timeout              INTEGER NOT NULL DEFAULT 0,
    cors                 INTEGER NOT NULL DEFAULT 0,
    expiry               INTEGER NOT NULL DEFAULT 0,
    actions              INTEGER NOT NULL DEFAULT 0,
    request_limit        INTEGER NOT NULL DEFAULT 0,
    description          TEXT NOT NULL DEFAULT '',
    listen               INTEGER NOT NULL DEFAULT 0,
    redirect             TEXT NOT NULL DEFAULT '',
    password             TEXT NOT NULL DEFAULT '',
    group_id             TEXT NOT NULL DEFAULT '',
    premium              INTEGER NOT NULL DEFAULT 1,
    created_at           TEXT NOT NULL,
    updated_at           TEXT NOT NULL,
    latest_request_at    TEXT
);

CREATE UNIQUE INDEX idx_tokens_alias ON tokens (alias) WHERE alias <> '';

CREATE TABLE requests (
    uuid                 TEXT PRIMARY KEY,
    token_id             TEXT NOT NULL REFERENCES tokens (uuid) ON DELETE CASCADE,
    type                 TEXT NOT NULL DEFAULT 'web',
    method               TEXT NOT NULL DEFAULT '',
    ip                   TEXT NOT NULL DEFAULT '',
    country              TEXT NOT NULL DEFAULT '',
    country_code         TEXT NOT NULL DEFAULT '',
    region               TEXT NOT NULL DEFAULT '',
    city                 TEXT NOT NULL DEFAULT '',
    hostname             TEXT NOT NULL DEFAULT '',
    user_agent           TEXT NOT NULL DEFAULT '',
    content              TEXT NOT NULL DEFAULT '',
    query                TEXT NOT NULL DEFAULT '{}',
    headers              TEXT NOT NULL DEFAULT '{}',
    url                  TEXT NOT NULL DEFAULT '',
    size                 BIGINT NOT NULL DEFAULT 0,
    sorting              BIGINT NOT NULL DEFAULT 0,
    custom_action_output TEXT NOT NULL DEFAULT '{}',
    custom_action_errors TEXT NOT NULL DEFAULT '{}',
    exec_time            DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at           TEXT NOT NULL
);

CREATE INDEX idx_requests_token ON requests (token_id, sorting DESC);

CREATE TABLE files (
    id           TEXT PRIMARY KEY,
    request_id   TEXT NOT NULL REFERENCES requests (uuid) ON DELETE CASCADE,
    filename     TEXT NOT NULL DEFAULT '',
    content_type TEXT NOT NULL DEFAULT '',
    size         BIGINT NOT NULL DEFAULT 0,
    path         TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_files_request ON files (request_id);
