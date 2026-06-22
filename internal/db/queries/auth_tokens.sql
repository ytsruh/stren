-- name: CreateAuthToken :one
-- Insert a new auth token row. The caller is responsible for hashing
-- the raw token with sha256 before storing. The raw token only ever
-- lives in the email link; the database never sees it.
INSERT INTO auth_tokens (id, user_id, purpose, token_hash, expires_at)
VALUES (?, ?, ?, ?, ?)
RETURNING id, user_id, purpose, token_hash, expires_at, used_at, created_at;

-- name: ConsumeAuthToken :one
-- Atomically claim a token: marks used_at and returns the row only
-- when the token exists for the given purpose, has not been used,
-- and has not expired. The single-statement UPDATE...RETURNING is
-- the concurrency primitive that prevents double-use across tabs,
-- devices, or racing requests.
--
-- `expires_at` comparison uses the literal `datetime('now')` rather
-- than a parameter so the database clock (not the application clock)
-- decides what "expired" means. SQLite's datetime('now') returns
-- UTC, matching the value written by the application.
UPDATE auth_tokens
SET used_at = datetime('now')
WHERE token_hash = ?
  AND purpose = ?
  AND used_at IS NULL
  AND expires_at > datetime('now')
RETURNING id, user_id, purpose, token_hash, expires_at, used_at, created_at;
