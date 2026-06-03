-- name: UpsertSummary :one
INSERT INTO summaries (id, contract_id, content)
VALUES ($1, $2, $3)
ON CONFLICT (contract_id) DO UPDATE SET content = EXCLUDED.content
RETURNING *;

-- name: GetSummaryByContract :one
SELECT * FROM summaries WHERE contract_id = $1;
