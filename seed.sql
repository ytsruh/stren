-- Seed script for strength tracker database
-- Run with: sqlite3 strength_tracker.db < seed.sql

-- Create tables if they don't exist
CREATE TABLE IF NOT EXISTS exercise_types (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS exercise_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    exercise_type_id INTEGER NOT NULL,
    reps INTEGER NOT NULL,
    weight REAL NOT NULL,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (exercise_type_id) REFERENCES exercise_types(id)
);

CREATE INDEX IF NOT EXISTS idx_entries_type ON exercise_entries(exercise_type_id);
CREATE INDEX IF NOT EXISTS idx_entries_created ON exercise_entries(created_at);

-- Insert exercises
INSERT OR IGNORE INTO exercise_types (name) VALUES
('Bench Press'),
('Squat'),
('Deadlift'),
('Overhead Press'),
('Barbell Row'),
('Pull Up'),
('Dips'),
('Lunges'),
('Romanian Deadlift'),
('Leg Press');
