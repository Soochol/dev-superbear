-- name: ListTimelineEventsByCase :many
SELECT * FROM timeline_events WHERE case_id = $1 ORDER BY date DESC;
-- name: CreateTimelineEvent :one
INSERT INTO timeline_events (case_id, date, type, title, content, ai_analysis, data)
VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *;
