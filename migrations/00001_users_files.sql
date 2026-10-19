-- +goose Up
CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email text NOT NULL UNIQUE CHECK (email <> ''),
    password_hash text NOT NULL CHECK (password_hash <> ''),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE files (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id uuid NOT NULL REFERENCES users(id),
    status text NOT NULL DEFAULT 'uploaded'
        CHECK (status IN ('uploaded', 'scanning', 'processing', 'ready', 'rejected')),
    original_name text NOT NULL CHECK (original_name <> ''),
    mime text NOT NULL CHECK (mime <> ''),
    size bigint NOT NULL CHECK (size >= 0),
    width integer CHECK (width > 0),
    height integer CHECK (height > 0),
    storage_key text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    CHECK ((width IS NULL) = (height IS NULL)),
    CHECK ((status = 'ready' AND storage_key LIKE 'final/%') OR
           (status <> 'ready' AND storage_key LIKE 'incoming/%'))
);
CREATE INDEX files_owner_created_idx ON files (owner_id, created_at DESC, id DESC)
    WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE files;
DROP TABLE users;
