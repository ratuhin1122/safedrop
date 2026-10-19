-- name: ListOwnerFiles :many
SELECT * FROM files WHERE owner_id = sqlc.arg(owner_id) AND deleted_at IS NULL
ORDER BY created_at DESC, id DESC LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);
