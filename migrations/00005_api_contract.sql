-- +goose Up
ALTER TABLE files ADD COLUMN reject_reason text;
UPDATE files SET reject_reason='processing_error' WHERE status='rejected';
ALTER TABLE share_links ADD COLUMN id uuid NOT NULL DEFAULT gen_random_uuid();
ALTER TABLE share_links ADD CONSTRAINT share_links_id_key UNIQUE (id);
CREATE TABLE upload_events (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX upload_events_user_time_idx ON upload_events(user_id, created_at);
INSERT INTO upload_events(user_id,created_at) SELECT owner_id,created_at FROM files;

-- +goose Down
DROP TABLE upload_events;
ALTER TABLE share_links DROP COLUMN id;
ALTER TABLE files DROP COLUMN reject_reason;
