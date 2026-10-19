-- name: ListExpiredShareFiles :many
SELECT DISTINCT file_id FROM share_links WHERE expires_at <= now()
ORDER BY file_id LIMIT 100;
