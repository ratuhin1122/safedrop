-- +goose Up
CREATE TABLE file_variants (
    file_id uuid NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    target_width integer NOT NULL CHECK (target_width IN (375, 768, 1200)),
    width integer NOT NULL CHECK (width > 0),
    height integer NOT NULL CHECK (height > 0),
    size bigint NOT NULL CHECK (size > 0),
    mime text NOT NULL CHECK (mime IN ('image/webp', 'image/avif')),
    storage_key text NOT NULL UNIQUE CHECK (storage_key LIKE 'final/%'),
    PRIMARY KEY (file_id, target_width)
);

-- Commit this outbox alongside ready status; delete originals after commit.
CREATE TABLE file_cleanup (
    file_id uuid PRIMARY KEY REFERENCES files(id) ON DELETE CASCADE,
    storage_key text NOT NULL CHECK (storage_key LIKE 'incoming/%')
);

-- +goose Down
DROP TABLE file_cleanup;
DROP TABLE file_variants;
