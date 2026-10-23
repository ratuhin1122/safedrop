-- name: RejectFile :execrows
UPDATE files SET status = 'rejected', reject_reason = COALESCE(reject_reason, 'processing_error')
WHERE id = sqlc.arg(id) AND status IN ('scanning', 'processing') AND deleted_at IS NULL;
