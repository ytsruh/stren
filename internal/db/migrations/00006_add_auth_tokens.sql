-- +goose Up
-- Auth tokens: short-lived, single-use credentials used to verify
-- ownership of an email address. Each row represents one outstanding
-- flow (password reset today; magic link / email verify tomorrow).
--
-- We never store the raw token: the application generates 32 random
-- bytes, base64url-encodes them, hashes with sha256, and stores the
-- hash. The raw token is only ever embedded in the email link.
-- A database leak therefore does not yield usable reset links.
--
-- The `purpose` column lets one table serve several flows without a
-- schema change per use case. New purposes are added by extending the
-- AuthTokenPurpose enum on the model side and adding new sqlc queries
-- filtered by the new value.
--
-- Atomic consumption: a single UPDATE marks `used_at = now` and
-- returns the user_id only when (purpose, token_hash, unused, not
-- expired) all match. RowsAffected on the UPDATE is the single
-- concurrency primitive that prevents double-use, even when a user
-- opens the reset link in two tabs at once.
CREATE TABLE auth_tokens (
    id         TEXT     PRIMARY KEY,
    user_id    TEXT     NOT NULL REFERENCES users(id),
    purpose    TEXT     NOT NULL,
    token_hash TEXT     NOT NULL,
    expires_at DATETIME NOT NULL,
    used_at    DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Lookup-by-hash-and-purpose is the hot path on every reset attempt
-- and every token consumption. Other columns are returned too, so
-- `purpose` is in the index for selectivity.
CREATE INDEX idx_auth_tokens_lookup ON auth_tokens(token_hash, purpose);

-- Listing a user's outstanding tokens (e.g. for the admin / future
-- "active sessions" UI) needs user_id.
CREATE INDEX idx_auth_tokens_user ON auth_tokens(user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_auth_tokens_user;
DROP INDEX IF EXISTS idx_auth_tokens_lookup;
DROP TABLE IF EXISTS auth_tokens;
