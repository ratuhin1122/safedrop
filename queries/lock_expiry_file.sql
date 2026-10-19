-- name: LockExpiryFile :one
SELECT * FROM files WHERE id = sqlc.arg(id) FOR UPDATE;
