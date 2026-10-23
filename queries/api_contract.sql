-- name: ListOwnerShares :many
SELECT s.id, s.file_id, s.expires_at, s.max_downloads, s.download_count,
       (s.password_hash IS NOT NULL)::boolean AS password_protected
FROM share_links s JOIN files f ON f.id=s.file_id
WHERE f.id=sqlc.arg(file_id) AND f.owner_id=sqlc.arg(owner_id) AND f.deleted_at IS NULL
ORDER BY s.expires_at, s.id;

-- name: DeleteOwnerShare :execrows
DELETE FROM share_links s USING files f
WHERE s.id=sqlc.arg(id) AND s.file_id=f.id AND f.owner_id=sqlc.arg(owner_id)
AND f.deleted_at IS NULL;

-- name: SetVirusRejection :exec
UPDATE files SET reject_reason='virus' WHERE id=$1 AND status='rejected';

-- name: OwnerUsage :one
SELECT COALESCE((SELECT bytes FROM upload_usage WHERE upload_usage.user_id=sqlc.arg(owner_id)
 AND day=(now() AT TIME ZONE 'UTC')::date),0)::bigint AS bytes_today,
 (SELECT count(*) FROM upload_events WHERE upload_events.user_id=sqlc.arg(owner_id)
 AND created_at >= now()-interval '1 hour')::bigint AS uploads_this_hour;

-- name: RecordUploadEvent :exec
INSERT INTO upload_events(user_id) VALUES ($1);
