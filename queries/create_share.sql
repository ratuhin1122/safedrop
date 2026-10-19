-- name: CreateShareLink :one
INSERT INTO share_links (token_hash, file_id, expires_at, password_hash, max_downloads)
SELECT sqlc.arg(token_hash), id, sqlc.arg(expires_at),
       sqlc.narg(password_hash), sqlc.narg(max_downloads)
FROM files WHERE id = sqlc.arg(file_id) AND owner_id = sqlc.arg(owner_id)
    AND status = 'ready' AND deleted_at IS NULL
RETURNING *;
