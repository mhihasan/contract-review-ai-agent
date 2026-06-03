-- name: CreateClauseAnalysis :one
INSERT INTO clause_analyses (id, clause_id, risk_level, explanation, ambiguous_language, recommendations, status)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetAnalysesByContract :many
SELECT ca.*
FROM clause_analyses ca
JOIN clauses c ON c.id = ca.clause_id
WHERE c.contract_id = $1;
