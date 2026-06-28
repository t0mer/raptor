-- Per-user ownership of capture URLs. Existing/open-mode tokens keep an empty
-- owner (accessible only in open mode or by admins).

ALTER TABLE tokens ADD COLUMN user_id TEXT NOT NULL DEFAULT '';

CREATE INDEX idx_tokens_user ON tokens (user_id);
