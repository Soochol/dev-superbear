-- name: ListTimelineEventsByCase :many
SELECT * FROM timeline_events WHERE case_id = $1 ORDER BY date ASC, created_at ASC;

-- name: ListTimelineEventsByType :many
SELECT * FROM timeline_events WHERE case_id = $1 AND type = $2 ORDER BY date ASC, created_at ASC;

-- name: ListTimelineEventsWithPaging :many
SELECT * FROM timeline_events WHERE case_id = $1 ORDER BY date ASC, created_at ASC LIMIT $2 OFFSET $3;

-- name: GetRecentTimelineEvents :many
SELECT * FROM timeline_events WHERE case_id = $1 ORDER BY date DESC, created_at DESC LIMIT $2;

-- name: CreateTimelineEvent :one
INSERT INTO timeline_events (case_id, date, day_offset, type, title, content, ai_analysis, data)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING *;

-- name: DeleteTimelineEventsByCase :exec
DELETE FROM timeline_events WHERE case_id = $1;
