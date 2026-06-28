-- Per-user ownership of capture URLs. (MySQL)

ALTER TABLE tokens ADD COLUMN user_id VARCHAR(64) NOT NULL DEFAULT '';

CREATE INDEX idx_tokens_user ON tokens (user_id);
