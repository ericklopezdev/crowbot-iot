-- name: UpsertInteractionAnalysis :one
INSERT INTO interaction_analysis (
    interaction_id, cognitive_area, question_type, complexity_level, sentiment, model
) VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (interaction_id) DO UPDATE
SET cognitive_area = EXCLUDED.cognitive_area,
    question_type = EXCLUDED.question_type,
    complexity_level = EXCLUDED.complexity_level,
    sentiment = EXCLUDED.sentiment,
    model = EXCLUDED.model,
    analyzed_at = now()
RETURNING *;

-- name: UpsertTopic :one
INSERT INTO topics (slug, label, area)
VALUES ($1, $2, $3)
ON CONFLICT (slug) DO UPDATE SET label = EXCLUDED.label
RETURNING *;

-- name: LinkInteractionTopic :exec
INSERT INTO interaction_topics (interaction_id, topic_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: BumpChildTopicStat :exec
INSERT INTO child_topic_stats (child_id, topic_id, interaction_count, first_seen_at, last_seen_at)
VALUES ($1, $2, 1, now(), now())
ON CONFLICT (child_id, topic_id) DO UPDATE
SET interaction_count = child_topic_stats.interaction_count + 1,
    last_seen_at = now();

-- name: ListChildTopicGraph :many
-- Interest-graph nodes: each topic the child engaged with + frequency.
SELECT t.id, t.slug, t.label, t.area, t.parent_id,
       s.interaction_count, s.last_seen_at
FROM child_topic_stats s
JOIN topics t ON t.id = s.topic_id
WHERE s.child_id = $1
ORDER BY s.interaction_count DESC;

-- name: AreaBreakdownByChild :many
-- Development by cognitive area (counts per area).
SELECT a.cognitive_area, count(*) AS interaction_count
FROM interaction_analysis a
JOIN interactions i ON i.id = a.interaction_id
WHERE i.child_id = $1 AND a.cognitive_area <> ''
GROUP BY a.cognitive_area
ORDER BY interaction_count DESC;

-- name: CreateRecommendation :one
INSERT INTO recommendations (child_id, topic_id, area, title, body)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListRecommendationsByChild :many
SELECT * FROM recommendations
WHERE child_id = $1 AND status <> 'dismissed'
ORDER BY generated_at DESC;
