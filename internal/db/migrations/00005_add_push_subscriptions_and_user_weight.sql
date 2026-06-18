-- +goose Up
-- Web push subscriptions: one row per (user, device). The endpoint is unique
-- across the table so a single device only ever has one record, even if it
-- re-subscribes after a service worker refresh.
CREATE TABLE push_subscriptions (
    id           TEXT     PRIMARY KEY,
    user_id      TEXT     NOT NULL REFERENCES users(id),
    endpoint     TEXT     UNIQUE NOT NULL,
    p256dh       TEXT     NOT NULL,
    auth         TEXT     NOT NULL,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_push_subs_user ON push_subscriptions(user_id);

-- User-level body weight preferences. target_weight is nullable so a user with
-- no goal simply has NULL; weight_unit defaults to kg to match the rest of
-- the app until per-user unit display is wired up.
ALTER TABLE users ADD COLUMN target_weight REAL;
ALTER TABLE users ADD COLUMN weight_unit  TEXT NOT NULL DEFAULT 'kg';

-- +goose Down
DROP INDEX IF EXISTS idx_push_subs_user;
DROP TABLE IF EXISTS push_subscriptions;
ALTER TABLE users DROP COLUMN weight_unit;
ALTER TABLE users DROP COLUMN target_weight;
