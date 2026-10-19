-- name: ListFileVariants :many
SELECT * FROM file_variants WHERE file_id = sqlc.arg(file_id) ORDER BY target_width;
