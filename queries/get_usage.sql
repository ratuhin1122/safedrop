-- name: GetUploadUsage :one
SELECT * FROM upload_usage WHERE user_id = sqlc.arg(user_id) AND day = sqlc.arg(day);
