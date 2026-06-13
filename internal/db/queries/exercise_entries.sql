-- name: CreateEntry :one
INSERT INTO exercise_entries (id, exercise_id, user_id, reps, weight, notes, rest_time, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id;

-- name: GetEntry :one
SELECT e.id, e.exercise_id, t.name as exercise_name, e.user_id, e.reps, e.weight, e.notes, e.rest_time, e.created_at
FROM exercise_entries e
JOIN exercises t ON e.exercise_id = t.id
WHERE e.id = ? AND e.user_id = ?;

-- name: UpdateEntry :exec
UPDATE exercise_entries
SET exercise_id = ?, reps = ?, weight = ?, notes = ?, rest_time = ?
WHERE id = ? AND user_id = ?;

-- name: UpdateEntryWithDate :exec
UPDATE exercise_entries
SET exercise_id = ?, reps = ?, weight = ?, notes = ?, rest_time = ?, created_at = ?
WHERE id = ? AND user_id = ?;

-- name: DeleteEntry :exec
DELETE FROM exercise_entries WHERE id = ? AND user_id = ?;

-- name: ListEntries :many
SELECT e.id, e.exercise_id, t.name as exercise_name, e.user_id, e.reps, e.weight, e.notes, e.rest_time, e.created_at
FROM exercise_entries e
JOIN exercises t ON e.exercise_id = t.id
WHERE e.user_id = ?
ORDER BY e.created_at DESC;

-- name: ListEntriesWithLimit :many
SELECT e.id, e.exercise_id, t.name as exercise_name, e.user_id, e.reps, e.weight, e.notes, e.rest_time, e.created_at
FROM exercise_entries e
JOIN exercises t ON e.exercise_id = t.id
WHERE e.user_id = ?
ORDER BY e.created_at DESC
LIMIT ?;

-- name: ListEntriesLast7Days :many
SELECT e.id, e.exercise_id, t.name as exercise_name, e.user_id, e.reps, e.weight, e.notes, e.rest_time, e.created_at
FROM exercise_entries e
JOIN exercises t ON e.exercise_id = t.id
WHERE e.created_at >= datetime('now', '-7 days') AND e.user_id = ?
ORDER BY e.created_at DESC;

-- name: GetEntriesByExercisePaginated :many
SELECT e.id, e.exercise_id, t.name as exercise_name, e.user_id, e.reps, e.weight, e.notes, e.rest_time, e.created_at
FROM exercise_entries e
JOIN exercises t ON e.exercise_id = t.id
WHERE e.exercise_id = ? AND e.user_id = ?
ORDER BY e.created_at DESC
LIMIT ? OFFSET ?;

-- name: GetMaxWeightByExercise :one
SELECT CAST(COALESCE(MAX(weight), 0) AS REAL) FROM exercise_entries
WHERE exercise_id = ? AND user_id = ?;

-- name: GetLastSetByExercise :one
SELECT e.id, e.exercise_id, t.name as exercise_name, e.user_id, e.reps, e.weight, e.notes, e.rest_time, e.created_at
FROM exercise_entries e
JOIN exercises t ON e.exercise_id = t.id
WHERE e.exercise_id = ? AND e.user_id = ?
ORDER BY e.created_at DESC
LIMIT 1;

-- name: GetEntriesByDateRange :many
SELECT e.id, e.exercise_id, t.name as exercise_name, e.user_id, e.reps, e.weight, e.notes, e.rest_time, e.created_at
FROM exercise_entries e
JOIN exercises t ON e.exercise_id = t.id
WHERE e.created_at BETWEEN ? AND ? AND e.user_id = ?
ORDER BY e.created_at DESC;
