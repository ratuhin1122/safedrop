-- name: CompleteFile :execrows
UPDATE files SET status = 'ready', storage_key = sqlc.arg(storage_key),
    mime = sqlc.arg(mime), size = sqlc.arg(size),
    width = sqlc.narg(width), height = sqlc.narg(height)
WHERE id = sqlc.arg(id) AND status = 'processing' AND deleted_at IS NULL;
