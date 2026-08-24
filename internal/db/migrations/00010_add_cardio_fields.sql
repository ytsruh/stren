-- +goose Up
-- Add cardio support to exercise entries. Strength entries keep using
-- reps/weight/rest_time; cardio entries use duration/distance (mandatory,
-- validated at the API layer) plus optional heart-rate and calories.
-- Columns are NOT NULL DEFAULT 0 so existing rows are unchanged and code
-- never has to handle NULL ("0 means not recorded").
ALTER TABLE exercise_entries ADD COLUMN duration_seconds INTEGER NOT NULL DEFAULT 0;
ALTER TABLE exercise_entries ADD COLUMN distance_meters REAL NOT NULL DEFAULT 0;
ALTER TABLE exercise_entries ADD COLUMN avg_heart_rate INTEGER NOT NULL DEFAULT 0;
ALTER TABLE exercise_entries ADD COLUMN calories_burned REAL NOT NULL DEFAULT 0;

-- Preferred display unit for cardio distances and pace ("km" or "mi").
-- Distances are stored in metres; this controls rendering only.
ALTER TABLE users ADD COLUMN distance_unit TEXT NOT NULL DEFAULT 'km';

-- +goose Down
ALTER TABLE users DROP COLUMN distance_unit;
ALTER TABLE exercise_entries DROP COLUMN calories_burned;
ALTER TABLE exercise_entries DROP COLUMN avg_heart_rate;
ALTER TABLE exercise_entries DROP COLUMN distance_meters;
ALTER TABLE exercise_entries DROP COLUMN duration_seconds;
