-- name: GetOwnerFile :one
SELECT * FROM files WHERE id = sqlc.arg(id) AND owner_id = sqlc.arg(owner_id) AND deleted_at IS NULL;

-- name: PageOwnerFiles :many
SELECT * FROM files WHERE owner_id = sqlc.arg(owner_id) AND deleted_at IS NULL
AND (NOT sqlc.arg(has_cursor)::boolean OR (created_at, id) < (sqlc.arg(cursor_time)::timestamptz, sqlc.arg(cursor_id)::uuid))
ORDER BY created_at DESC, id DESC LIMIT sqlc.arg(page_size);
