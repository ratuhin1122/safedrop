-- +goose Up
CREATE TABLE share_links (
    token_hash bytea PRIMARY KEY CHECK (octet_length(token_hash) = 32),
    file_id uuid NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    password_hash text CHECK (password_hash IS NULL OR password_hash <> ''),
    max_downloads bigint CHECK (max_downloads > 0),
    download_count bigint NOT NULL DEFAULT 0 CHECK (download_count >= 0),
    CHECK (max_downloads IS NULL OR download_count <= max_downloads)
);
CREATE INDEX share_links_file_idx ON share_links (file_id);
CREATE INDEX share_links_expiry_idx ON share_links (expires_at);

CREATE TABLE upload_usage (
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    day date NOT NULL,
    bytes bigint NOT NULL DEFAULT 0 CHECK (bytes >= 0),
    count bigint NOT NULL DEFAULT 0 CHECK (count >= 0),
    PRIMARY KEY (user_id, day)
);

-- +goose Down
DROP TABLE upload_usage;
DROP TABLE share_links;
