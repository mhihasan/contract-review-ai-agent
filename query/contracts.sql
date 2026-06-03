-- name: CreateContract :one
INSERT INTO contracts (id, filename, raw_text, status)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetContract :one
SELECT * FROM contracts WHERE id = $1;

-- name: UpdateContractStatus :exec
UPDATE contracts SET status = $2 WHERE id = $1;
