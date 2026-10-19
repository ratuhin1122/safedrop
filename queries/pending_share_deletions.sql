-- name: ListShareFileDeletions :many
SELECT file_id FROM share_file_deletions ORDER BY attempted_at, file_id LIMIT 100;

-- name: TouchShareFileDeletion :exec
UPDATE share_file_deletions SET attempted_at = now() WHERE file_id = sqlc.arg(file_id);
