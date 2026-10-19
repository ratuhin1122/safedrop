-- name: ChargeUpload :one
INSERT INTO upload_usage (user_id, day, bytes, count)
SELECT sqlc.arg(user_id)::uuid, (now() AT TIME ZONE 'UTC')::date, sqlc.arg(bytes)::bigint, 1
WHERE sqlc.arg(bytes)::bigint <= sqlc.arg(quota_bytes)::bigint
ON CONFLICT (user_id, day) DO UPDATE
SET bytes = upload_usage.bytes + EXCLUDED.bytes, count = upload_usage.count + 1
WHERE upload_usage.bytes <= sqlc.arg(quota_bytes)::bigint - EXCLUDED.bytes
RETURNING *;
