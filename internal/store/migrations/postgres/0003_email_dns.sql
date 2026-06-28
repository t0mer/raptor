-- Phase 3: email- and DNS-capture extras on the requests table. (PostgreSQL)

ALTER TABLE requests ADD COLUMN sender       TEXT NOT NULL DEFAULT '';
ALTER TABLE requests ADD COLUMN message_id   TEXT NOT NULL DEFAULT '';
ALTER TABLE requests ADD COLUMN destinations TEXT NOT NULL DEFAULT '';
ALTER TABLE requests ADD COLUMN subject      TEXT NOT NULL DEFAULT '';
ALTER TABLE requests ADD COLUMN text_content TEXT NOT NULL DEFAULT '';
ALTER TABLE requests ADD COLUMN checks       TEXT NOT NULL DEFAULT '{}';
