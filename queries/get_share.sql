-- name: GetShareLink :one
SELECT s.* FROM share_links s JOIN files f ON f.id = s.file_id
WHERE s.token_hash = sqlc.arg(token_hash) AND s.expires_at > now()
    AND (s.max_downloads IS NULL OR s.download_count < s.max_downloads)
    AND f.status = 'ready' AND f.deleted_at IS NULL;
