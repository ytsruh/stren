-- name: CreateGoal :one
INSERT INTO goals (id, user_id, title, description, start_date, target_date, end_date, completed_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetGoal :one
SELECT * FROM goals
WHERE id = ? AND user_id = ?;

-- name: ListActiveGoals :many
-- Active goals: completed_at IS NULL. Order by target_date asc with nulls
-- last, then by created_at asc as a stable tiebreaker. The CASE expression
-- emulates NULLS LAST for SQLite versions that don't support it natively.
SELECT * FROM goals
WHERE user_id = ? AND completed_at IS NULL
ORDER BY CASE WHEN target_date IS NULL THEN 1 ELSE 0 END, target_date ASC, created_at ASC;

-- name: ListCompletedGoals :many
-- Completed goals: completed_at IS NOT NULL. Most recently completed first.
SELECT * FROM goals
WHERE user_id = ? AND completed_at IS NOT NULL
ORDER BY completed_at DESC;

-- name: UpdateGoal :exec
-- Update the editable fields. completed_at is managed by MarkGoalComplete
-- and ReopenGoal (single-purpose methods that don't touch the rest of the
-- row), so the edit form cannot accidentally clear or set the completed
-- state. updated_at is bumped automatically.
UPDATE goals
SET title = ?,
    description = ?,
    start_date = ?,
    target_date = ?,
    end_date = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND user_id = ?;

-- name: MarkGoalComplete :exec
-- Atomically set completed_at and bump updated_at. Scoped to user_id so
-- the request cannot mark another user's goal complete.
UPDATE goals
SET completed_at = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND user_id = ? AND completed_at IS NULL;

-- name: ReopenGoal :exec
-- Atomically clear completed_at and bump updated_at. No-op if the goal
-- is already active (completed_at is already NULL), so the route can
-- call it without first checking the current state.
UPDATE goals
SET completed_at = NULL, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND user_id = ?;

-- name: DeleteGoal :exec
DELETE FROM goals
WHERE id = ? AND user_id = ?;
