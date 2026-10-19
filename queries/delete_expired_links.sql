-- name: DeleteExpiredFileLinks :exec
DELETE FROM share_links WHERE file_id = sqlc.arg(file_id) AND expires_at <= now();

-- name: HasActiveShareLinks :one
SELECT EXISTS(SELECT 1 FROM share_links WHERE file_id = sqlc.arg(file_id) AND expires_at > now());
