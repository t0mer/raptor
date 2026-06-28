-- Phase 4: Custom Actions engine — action definitions and per-request run logs. (MySQL)

CREATE TABLE actions (
    uuid        VARCHAR(64) PRIMARY KEY,
    token_id    VARCHAR(64) NOT NULL,
    type        VARCHAR(64) NOT NULL,
    position    INT NOT NULL DEFAULT 0,
    name        VARCHAR(255) NOT NULL DEFAULT '',
    disabled    TINYINT NOT NULL DEFAULT 0,
    parameters  MEDIUMTEXT NOT NULL DEFAULT ('{}'),
    queue       TINYINT NOT NULL DEFAULT 0,
    delay       INT NOT NULL DEFAULT 0,
    "condition" VARCHAR(64) NOT NULL DEFAULT '',
    created_at  VARCHAR(64) NOT NULL,
    updated_at  VARCHAR(64) NOT NULL,
    CONSTRAINT fk_actions_token FOREIGN KEY (token_id) REFERENCES tokens (uuid) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_actions_token ON actions (token_id, position);

CREATE TABLE action_runs (
    id          VARCHAR(64) PRIMARY KEY,
    request_id  VARCHAR(64) NOT NULL,
    action_id   VARCHAR(64) NOT NULL,
    action_type VARCHAR(64) NOT NULL,
    action_name VARCHAR(255) NOT NULL DEFAULT '',
    position    INT NOT NULL DEFAULT 0,
    output      LONGTEXT NOT NULL DEFAULT (''),
    error       LONGTEXT NOT NULL DEFAULT (''),
    created_at  VARCHAR(64) NOT NULL,
    CONSTRAINT fk_action_runs_request FOREIGN KEY (request_id) REFERENCES requests (uuid) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_action_runs_request ON action_runs (request_id, position);
