-- +goose Up
CREATE TABLE share_file_deletions (
    file_id uuid PRIMARY KEY REFERENCES files(id) ON DELETE CASCADE,
    attempted_at timestamptz NOT NULL DEFAULT 'epoch'
);

-- +goose Down
DROP TABLE share_file_deletions;
