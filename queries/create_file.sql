-- name: CreateFile :one
INSERT INTO files (owner_id, original_name, mime, size, storage_key)
VALUES (sqlc.arg(owner_id), sqlc.arg(original_name), sqlc.arg(mime),
        sqlc.arg(size), sqlc.arg(storage_key)) RETURNING *;
