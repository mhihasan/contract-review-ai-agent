-- name: CreateClause :one
INSERT INTO clauses (id, contract_id, sequence_number, text)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetClausesByContract :many
SELECT * FROM clauses WHERE contract_id = $1 ORDER BY sequence_number;
