-- name: ListFileCleanup :many
SELECT * FROM file_cleanup WHERE file_id > sqlc.arg(after_id)
ORDER BY file_id LIMIT 100;
