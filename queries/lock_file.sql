-- name: LockFile :one
SELECT * FROM files WHERE id = sqlc.arg(id) AND deleted_at IS NULL FOR UPDATE;
