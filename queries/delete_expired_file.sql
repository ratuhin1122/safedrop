-- name: DeleteExpiredFile :exec
DELETE FROM files WHERE id = sqlc.arg(id) AND deleted_at IS NOT NULL
AND EXISTS(SELECT 1 FROM share_file_deletions WHERE file_id = files.id);
