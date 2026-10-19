-- name: ScheduleFileCleanup :exec
INSERT INTO file_cleanup (file_id, storage_key) VALUES (sqlc.arg(file_id), sqlc.arg(storage_key))
ON CONFLICT (file_id) DO NOTHING;
