-- name: CreateEntry :one
INSERT INTO exercise_entries (exercise_type_id, reps, weight, notes, created_at)
VALUES (?, ?, ?, ?, ?)
RETURNING id;

-- name: GetEntry :one
SELECT e.id, e.exercise_type_id, t.name as exercise_name, e.reps, e.weight, e.notes, e.created_at
FROM exercise_entries e
JOIN exercise_types t ON e.exercise_type_id = t.id
WHERE e.id = ?;

-- name: UpdateEntry :exec
UPDATE exercise_entries
SET exercise_type_id = ?, reps = ?, weight = ?, notes = ?
WHERE id = ?;

-- name: UpdateEntryWithDate :exec
UPDATE exercise_entries
SET exercise_type_id = ?, reps = ?, weight = ?, notes = ?, created_at = ?
WHERE id = ?;

-- name: DeleteEntry :exec
DELETE FROM exercise_entries WHERE id = ?;

-- name: ListEntries :many
SELECT e.id, e.exercise_type_id, t.name as exercise_name, e.reps, e.weight, e.notes, e.created_at
FROM exercise_entries e
JOIN exercise_types t ON e.exercise_type_id = t.id
ORDER BY e.created_at DESC;

-- name: ListEntriesWithLimit :many
SELECT e.id, e.exercise_type_id, t.name as exercise_name, e.reps, e.weight, e.notes, e.created_at
FROM exercise_entries e
JOIN exercise_types t ON e.exercise_type_id = t.id
ORDER BY e.created_at DESC
LIMIT ?;

-- name: GetEntriesByExercise :many
SELECT e.id, e.exercise_type_id, t.name as exercise_name, e.reps, e.weight, e.notes, e.created_at
FROM exercise_entries e
JOIN exercise_types t ON e.exercise_type_id = t.id
WHERE t.name = ?
ORDER BY e.created_at DESC;

-- name: GetEntriesByDateRange :many
SELECT e.id, e.exercise_type_id, t.name as exercise_name, e.reps, e.weight, e.notes, e.created_at
FROM exercise_entries e
JOIN exercise_types t ON e.exercise_type_id = t.id
WHERE e.created_at BETWEEN ? AND ?
ORDER BY e.created_at DESC;
