-- name: GetShareLinkState :one
SELECT * FROM share_links WHERE token_hash = sqlc.arg(token_hash);
