-- name: AdvanceFile :execrows
UPDATE files SET status = sqlc.arg(next_status), reject_reason = CASE
 WHEN sqlc.arg(next_status)::text = 'rejected' THEN COALESCE(reject_reason, 'processing_error')
 ELSE reject_reason END
WHERE id = sqlc.arg(id) AND status = sqlc.arg(expected_status) AND deleted_at IS NULL
AND ((status = 'uploaded' AND sqlc.arg(next_status)::text = 'scanning')
  OR (status = 'scanning' AND sqlc.arg(next_status)::text IN ('processing', 'rejected'))
  OR (status = 'processing' AND sqlc.arg(next_status)::text = 'rejected'));
