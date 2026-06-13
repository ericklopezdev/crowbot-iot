-- name: CreateInteraction :one
INSERT INTO interactions (
    device_id, child_id, session_id, transcript, response_text,
    stt_latency_ms, llm_latency_ms, tts_latency_ms, total_latency_ms,
    llm_model, llm_tokens_in, llm_tokens_out
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
RETURNING *;

-- name: ListInteractionsByChild :many
SELECT * FROM interactions
WHERE child_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetInteraction :one
SELECT * FROM interactions WHERE id = $1;

-- name: ListUnanalyzedInteractions :many
-- Feeds the analysis worker: interactions without an analysis row yet.
SELECT i.* FROM interactions i
LEFT JOIN interaction_analysis a ON a.interaction_id = i.id
WHERE a.interaction_id IS NULL
ORDER BY i.created_at
LIMIT $1;

-- name: ChildUsageByHour :many
-- Days x hours usage for the dashboard heatmap.
SELECT
    date_trunc('hour', created_at) AS bucket,
    count(*)                       AS interaction_count
FROM interactions
WHERE child_id = $1 AND created_at >= $2 AND created_at < $3
GROUP BY bucket
ORDER BY bucket;
