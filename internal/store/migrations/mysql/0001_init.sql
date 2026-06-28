-- Phase 1 core schema: tokens, requests, files. (MySQL 8.0.13+)
-- Reserved words "groups"/"condition" are double-quoted; the connection runs in
-- ANSI_QUOTES mode (see store.mysqlDSN). LONGTEXT/MEDIUMTEXT use parenthesized
-- expression defaults, which require MySQL 8.0.13 or newer.

CREATE TABLE tokens (
    uuid                 VARCHAR(64) PRIMARY KEY,
    alias                VARCHAR(255) NOT NULL DEFAULT '',
    default_status       INT NOT NULL DEFAULT 200,
    default_content      LONGTEXT NOT NULL DEFAULT (''),
    default_content_type VARCHAR(255) NOT NULL DEFAULT 'text/plain',
    timeout              INT NOT NULL DEFAULT 0,
    cors                 TINYINT NOT NULL DEFAULT 0,
    expiry               INT NOT NULL DEFAULT 0,
    actions              TINYINT NOT NULL DEFAULT 0,
    request_limit        INT NOT NULL DEFAULT 0,
    description          VARCHAR(1024) NOT NULL DEFAULT '',
    listen               INT NOT NULL DEFAULT 0,
    redirect             VARCHAR(2048) NOT NULL DEFAULT '',
    password             VARCHAR(512) NOT NULL DEFAULT '',
    group_id             VARCHAR(64) NOT NULL DEFAULT '',
    premium              TINYINT NOT NULL DEFAULT 1,
    created_at           VARCHAR(64) NOT NULL,
    updated_at           VARCHAR(64) NOT NULL,
    latest_request_at    VARCHAR(64)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_tokens_alias ON tokens (alias);

CREATE TABLE requests (
    uuid                 VARCHAR(64) PRIMARY KEY,
    token_id             VARCHAR(64) NOT NULL,
    type                 VARCHAR(32) NOT NULL DEFAULT 'web',
    method               VARCHAR(32) NOT NULL DEFAULT '',
    ip                   VARCHAR(64) NOT NULL DEFAULT '',
    country              VARCHAR(255) NOT NULL DEFAULT '',
    country_code         VARCHAR(8) NOT NULL DEFAULT '',
    region               VARCHAR(255) NOT NULL DEFAULT '',
    city                 VARCHAR(255) NOT NULL DEFAULT '',
    hostname             VARCHAR(512) NOT NULL DEFAULT '',
    user_agent           VARCHAR(512) NOT NULL DEFAULT '',
    content              LONGTEXT NOT NULL DEFAULT (''),
    query                MEDIUMTEXT NOT NULL DEFAULT ('{}'),
    headers              MEDIUMTEXT NOT NULL DEFAULT ('{}'),
    url                  VARCHAR(2048) NOT NULL DEFAULT '',
    size                 BIGINT NOT NULL DEFAULT 0,
    sorting              BIGINT NOT NULL DEFAULT 0,
    custom_action_output MEDIUMTEXT NOT NULL DEFAULT ('{}'),
    custom_action_errors MEDIUMTEXT NOT NULL DEFAULT ('{}'),
    exec_time            DOUBLE NOT NULL DEFAULT 0,
    created_at           VARCHAR(64) NOT NULL,
    CONSTRAINT fk_requests_token FOREIGN KEY (token_id) REFERENCES tokens (uuid) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_requests_token ON requests (token_id, sorting DESC);

CREATE TABLE files (
    id           VARCHAR(64) PRIMARY KEY,
    request_id   VARCHAR(64) NOT NULL,
    filename     VARCHAR(512) NOT NULL DEFAULT '',
    content_type VARCHAR(255) NOT NULL DEFAULT '',
    size         BIGINT NOT NULL DEFAULT 0,
    path         VARCHAR(1024) NOT NULL DEFAULT '',
    CONSTRAINT fk_files_request FOREIGN KEY (request_id) REFERENCES requests (uuid) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_files_request ON files (request_id);
