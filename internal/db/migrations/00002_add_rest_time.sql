-- +goose Up
ALTER TABLE exercise_entries ADD COLUMN rest_time INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE exercise_entries DROP COLUMN rest_time;