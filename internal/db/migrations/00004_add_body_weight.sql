-- +goose Up
CREATE TABLE weight_entries (
    id         TEXT    PRIMARY KEY,
    user_id    TEXT    NOT NULL REFERENCES users(id),
    weight     REAL    NOT NULL,
    notes      TEXT,
    photo_key  TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_weight_entries_user    ON weight_entries(user_id);
CREATE INDEX idx_weight_entries_created ON weight_entries(created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_weight_entries_created;
DROP INDEX IF EXISTS idx_weight_entries_user;
DROP TABLE IF EXISTS weight_entries;
