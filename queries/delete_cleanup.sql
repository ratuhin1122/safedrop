-- name: DeleteFileCleanup :exec
DELETE FROM file_cleanup WHERE file_id = sqlc.arg(file_id) AND storage_key = sqlc.arg(storage_key);
