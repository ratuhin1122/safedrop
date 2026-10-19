-- name: AddUploadUsage :one
INSERT INTO upload_usage (user_id, day, bytes, count)
VALUES (sqlc.arg(user_id), sqlc.arg(day), sqlc.arg(bytes), 1)
ON CONFLICT (user_id, day) DO UPDATE
SET bytes = upload_usage.bytes + EXCLUDED.bytes, count = upload_usage.count + 1
RETURNING *;
