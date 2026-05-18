-- name: CreateEntry :one
INSERT INTO exercise_entries (exercise_id, user_id, reps, weight, notes, created_at)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING id;

-- name: GetEntry :one
SELECT e.id, e.exercise_id, t.name as exercise_name, e.user_id, e.reps, e.weight, e.notes, e.created_at
FROM exercise_entries e
JOIN exercises t ON e.exercise_id = t.id
WHERE e.id = ? AND e.user_id = ?;

-- name: UpdateEntry :exec
UPDATE exercise_entries
SET exercise_id = ?, reps = ?, weight = ?, notes = ?
WHERE id = ? AND user_id = ?;

-- name: UpdateEntryWithDate :exec
UPDATE exercise_entries
SET exercise_id = ?, reps = ?, weight = ?, notes = ?, created_at = ?
WHERE id = ? AND user_id = ?;

-- name: DeleteEntry :exec
DELETE FROM exercise_entries WHERE id = ? AND user_id = ?;

-- name: ListEntries :many
SELECT e.id, e.exercise_id, t.name as exercise_name, e.user_id, e.reps, e.weight, e.notes, e.created_at
FROM exercise_entries e
JOIN exercises t ON e.exercise_id = t.id
WHERE e.user_id = ?
ORDER BY e.created_at DESC;

-- name: ListEntriesWithLimit :many
SELECT e.id, e.exercise_id, t.name as exercise_name, e.user_id, e.reps, e.weight, e.notes, e.created_at
FROM exercise_entries e
JOIN exercises t ON e.exercise_id = t.id
WHERE e.user_id = ?
ORDER BY e.created_at DESC
LIMIT ?;

-- name: ListEntriesLast30Days :many
SELECT e.id, e.exercise_id, t.name as exercise_name, e.user_id, e.reps, e.weight, e.notes, e.created_at
FROM exercise_entries e
JOIN exercises t ON e.exercise_id = t.id
WHERE e.created_at >= datetime('now', '-30 days') AND e.user_id = ?
ORDER BY e.created_at DESC;

-- name: GetEntriesByExercise :many
SELECT e.id, e.exercise_id, t.name as exercise_name, e.user_id, e.reps, e.weight, e.notes, e.created_at
FROM exercise_entries e
JOIN exercises t ON e.exercise_id = t.id
WHERE t.name = ? AND e.user_id = ?
ORDER BY e.created_at DESC;

-- name: GetEntriesByDateRange :many
SELECT e.id, e.exercise_id, t.name as exercise_name, e.user_id, e.reps, e.weight, e.notes, e.created_at
FROM exercise_entries e
JOIN exercises t ON e.exercise_id = t.id
WHERE e.created_at BETWEEN ? AND ? AND e.user_id = ?
ORDER BY e.created_at DESC;
