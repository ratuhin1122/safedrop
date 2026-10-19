-- name: GetExpiredFile :one
SELECT f.* FROM files f JOIN share_file_deletions d ON d.file_id = f.id
WHERE f.id = sqlc.arg(id) AND f.deleted_at IS NOT NULL;
