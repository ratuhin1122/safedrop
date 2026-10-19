-- name: GetFileCleanup :one
SELECT * FROM file_cleanup WHERE file_id = sqlc.arg(file_id);
