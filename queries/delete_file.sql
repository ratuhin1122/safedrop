-- name: SoftDeleteFile :execrows
UPDATE files SET deleted_at = now()
WHERE id = sqlc.arg(id) AND owner_id = sqlc.arg(owner_id) AND deleted_at IS NULL;
