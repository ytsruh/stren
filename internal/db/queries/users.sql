-- name: CreateUser :one
INSERT INTO users (id, name, email, password_hash, is_admin)
VALUES (?, ?, ?, ?, ?)
RETURNING id;

-- name: GetUserByEmail :one
SELECT id, name, email, password_hash, is_admin, target_weight, weight_unit, reminder_enabled, reminder_frequency, reminder_day_of_week, reminder_time, reminder_email_enabled, reminder_push_enabled, reminder_next_fire_at, reminder_last_fired_at, created_at, updated_at
FROM users
WHERE email = ?;

-- name: GetUserByID :one
SELECT id, name, email, password_hash, is_admin, target_weight, weight_unit, reminder_enabled, reminder_frequency, reminder_day_of_week, reminder_time, reminder_email_enabled, reminder_push_enabled, reminder_next_fire_at, reminder_last_fired_at, created_at, updated_at
FROM users
WHERE id = ?;

-- name: ListUsers :many
SELECT id, name, email, password_hash, is_admin, target_weight, weight_unit, reminder_enabled, reminder_frequency, reminder_day_of_week, reminder_time, reminder_email_enabled, reminder_push_enabled, reminder_next_fire_at, reminder_last_fired_at, created_at, updated_at
FROM users
ORDER BY created_at DESC;

-- name: UpdateUser :exec
UPDATE users
SET name = ?,
    target_weight = ?,
    weight_unit = ?,
    updated_at = ?
WHERE id = ?;

-- name: UpdateUserPassword :exec
-- Replace a user's password hash. Used by the password-reset flow
-- after a reset token has been successfully consumed. Separate from
-- UpdateUser so the profile-editing form cannot be tricked into
-- clearing the password by omitting fields.
UPDATE users
SET password_hash = ?,
    updated_at = ?
WHERE id = ?;

-- name: UpdateUserReminder :exec
-- Replace a user's reminder preferences and bump updated_at so the
-- row is re-evaluated by the periodic tick on the next run. The day
-- of week is nullable because off / daily frequencies don't need it;
-- every other parameter is required and is asserted by the controller
-- before the call. next_fire_at is recomputed by the caller and passed
-- in as a parameter; we do not derive it here so the same query works
-- for both "user changed preferences" and "tick just fired" callers.
UPDATE users
SET reminder_enabled        = ?,
    reminder_frequency      = ?,
    reminder_day_of_week    = ?,
    reminder_time           = ?,
    reminder_email_enabled  = ?,
    reminder_next_fire_at   = ?,
    updated_at              = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: ListUsersDueForReminder :many
-- Returns every enabled user whose next_fire_at is at or before the
-- supplied reference time. The hourly tick calls this once per hour;
-- the indexed next_fire_at column keeps the scan small even when the
-- users table grows. We pull back the full row so the orchestrator can
-- build its email payload without an extra round-trip.
SELECT id, name, email, password_hash, is_admin, target_weight, weight_unit, reminder_enabled, reminder_frequency, reminder_day_of_week, reminder_time, reminder_email_enabled, reminder_push_enabled, reminder_next_fire_at, reminder_last_fired_at, created_at, updated_at
FROM users
WHERE reminder_enabled = 1
  AND reminder_next_fire_at IS NOT NULL
  AND reminder_next_fire_at <= ?;

-- name: MarkUserReminderFired :exec
-- Atomically set last_fired_at = now and advance next_fire_at to the
-- caller's computed next occurrence. Scoped to id so a misbehaving
-- caller cannot advance another user's row. updated_at is bumped so
-- the row's edit history stays accurate (useful for future "when was
-- this user last updated" UI).
UPDATE users
SET reminder_last_fired_at = ?,
    reminder_next_fire_at  = ?,
    updated_at             = CURRENT_TIMESTAMP
WHERE id = ?;
