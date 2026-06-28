-- Phase 2: token groups for organising capture URLs. (MySQL)

CREATE TABLE "groups" (
    id         VARCHAR(64) PRIMARY KEY,
    name       VARCHAR(255) NOT NULL DEFAULT '',
    color      VARCHAR(32) NOT NULL DEFAULT '',
    created_at VARCHAR(64) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
