-- Per-user ownership of capture URLs. (PostgreSQL)

ALTER TABLE tokens ADD COLUMN user_id TEXT NOT NULL DEFAULT '';

CREATE INDEX idx_tokens_user ON tokens (user_id);
