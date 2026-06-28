-- Phase 3: email- and DNS-capture extras on the requests table. (MySQL)

ALTER TABLE requests ADD COLUMN sender       VARCHAR(512) NOT NULL DEFAULT '';
ALTER TABLE requests ADD COLUMN message_id   VARCHAR(512) NOT NULL DEFAULT '';
ALTER TABLE requests ADD COLUMN destinations VARCHAR(1024) NOT NULL DEFAULT '';
ALTER TABLE requests ADD COLUMN subject      VARCHAR(998) NOT NULL DEFAULT '';
ALTER TABLE requests ADD COLUMN text_content LONGTEXT NOT NULL DEFAULT ('');
ALTER TABLE requests ADD COLUMN checks       MEDIUMTEXT NOT NULL DEFAULT ('{}');
