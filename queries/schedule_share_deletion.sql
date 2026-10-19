-- name: MarkFileExpired :exec
UPDATE files SET deleted_at = COALESCE(deleted_at, now()) WHERE id = sqlc.arg(id);

-- name: ScheduleShareFileDeletion :exec
INSERT INTO share_file_deletions (file_id) VALUES (sqlc.arg(file_id)) ON CONFLICT DO NOTHING;
