-- Phase 2: token groups for organising capture URLs.

CREATE TABLE groups (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL DEFAULT '',
    color      TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);
