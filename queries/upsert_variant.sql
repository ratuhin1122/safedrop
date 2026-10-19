-- name: UpsertFileVariant :exec
INSERT INTO file_variants (file_id, target_width, width, height, size, mime, storage_key)
VALUES (sqlc.arg(file_id), sqlc.arg(target_width), sqlc.arg(width), sqlc.arg(height),
    sqlc.arg(size), sqlc.arg(mime), sqlc.arg(storage_key))
ON CONFLICT (file_id, target_width) DO UPDATE SET width = EXCLUDED.width,
    height = EXCLUDED.height, size = EXCLUDED.size, mime = EXCLUDED.mime,
    storage_key = EXCLUDED.storage_key;
