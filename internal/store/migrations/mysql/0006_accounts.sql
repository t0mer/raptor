-- Accounts & org: users, API keys, and sessions. (MySQL)
-- The default utf8mb4 collation is case-insensitive, so a plain UNIQUE index on
-- email enforces case-insensitive uniqueness.

CREATE TABLE users (
    id            VARCHAR(64) PRIMARY KEY,
    email         VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL DEFAULT '',
    role          VARCHAR(32) NOT NULL DEFAULT 'user',
    created_at    VARCHAR(64) NOT NULL,
    updated_at    VARCHAR(64) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE UNIQUE INDEX idx_users_email ON users (email);

CREATE TABLE api_keys (
    id           VARCHAR(64) PRIMARY KEY,
    user_id      VARCHAR(64) NOT NULL,
    name         VARCHAR(255) NOT NULL DEFAULT '',
    key_hash     VARCHAR(255) NOT NULL,
    last_used_at VARCHAR(64),
    created_at   VARCHAR(64) NOT NULL,
    CONSTRAINT fk_api_keys_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE UNIQUE INDEX idx_api_keys_hash ON api_keys (key_hash);
CREATE INDEX idx_api_keys_user ON api_keys (user_id);

CREATE TABLE sessions (
    id         VARCHAR(64) PRIMARY KEY,
    user_id    VARCHAR(64) NOT NULL,
    expires_at VARCHAR(64) NOT NULL,
    created_at VARCHAR(64) NOT NULL,
    CONSTRAINT fk_sessions_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_sessions_user ON sessions (user_id);
