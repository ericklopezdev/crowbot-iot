package store

// Hand-written (non-generated) dashboard reads. These avoid a sqlc type-inference
// quirk: for `date_trunc('hour', created_at)` sqlc inferred the bucket as
// pgtype.Interval, which fails to scan a timestamptz. Here the bucket is cast and
// scanned as time.Time.

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// UsageBucket is one hour bucket of the usage heatmap.
type UsageBucket struct {
	Bucket           time.Time `json:"bucket"`
	InteractionCount int64     `json:"interaction_count"`
}

const childUsageByHourSQL = `
SELECT date_trunc('hour', created_at)::timestamptz AS bucket, count(*) AS interaction_count
FROM interactions
WHERE child_id = $1 AND created_at >= $2 AND created_at < $3
GROUP BY bucket
ORDER BY bucket`

// ChildUsage returns per-hour interaction counts for a child in [from, to).
func (q *Queries) ChildUsage(ctx context.Context, childID uuid.UUID, from, to time.Time) ([]UsageBucket, error) {
	rows, err := q.db.Query(ctx, childUsageByHourSQL, childID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UsageBucket
	for rows.Next() {
		var b UsageBucket
		if err := rows.Scan(&b.Bucket, &b.InteractionCount); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// CountInteractionsByChild is the total interaction count for a child.
func (q *Queries) CountInteractionsByChild(ctx context.Context, childID uuid.UUID) (int64, error) {
	var n int64
	err := q.db.QueryRow(ctx,
		`SELECT count(*) FROM interactions WHERE child_id = $1`, childID).Scan(&n)
	return n, err
}
