-- Caller must verify any password before reserving a download.
-- name: ConsumeShareDownload :one
UPDATE share_links s SET download_count = s.download_count + 1
WHERE s.token_hash = sqlc.arg(token_hash) AND s.expires_at > now()
    AND (s.max_downloads IS NULL OR s.download_count < s.max_downloads)
    AND EXISTS (SELECT 1 FROM files f WHERE f.id = s.file_id
        AND f.status = 'ready' AND f.deleted_at IS NULL)
RETURNING s.*;
