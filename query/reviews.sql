-- name: CreateReview :one
INSERT INTO reviews (id, clause_id, decision, annotation)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetReviewsByContract :many
SELECT r.*
FROM reviews r
JOIN clauses c ON c.id = r.clause_id
WHERE c.contract_id = $1;
