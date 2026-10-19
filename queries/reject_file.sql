-- name: RejectFile :execrows
UPDATE files SET status = 'rejected'
WHERE id = sqlc.arg(id) AND status IN ('scanning', 'processing') AND deleted_at IS NULL;
