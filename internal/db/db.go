// Package db provides a wrapper around SQLite database operations.
// All external database dependencies are isolated here for easy replacement.
package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps the sql.DB to provide application-specific operations
type DB struct {
	conn *sql.DB
}

// New creates a new database connection with migrations applied
func New(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (d *DB) Close() error {
	return d.conn.Close()
}

// migrate runs database migrations
func (d *DB) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS exercise_types (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS exercise_entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			exercise_type_id INTEGER NOT NULL,
			reps INTEGER NOT NULL,
			weight REAL NOT NULL,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (exercise_type_id) REFERENCES exercise_types(id)
		)`,

		`CREATE INDEX IF NOT EXISTS idx_entries_type ON exercise_entries(exercise_type_id)`,
		`CREATE INDEX IF NOT EXISTS idx_entries_created ON exercise_entries(created_at)`,
	}

	for _, migration := range migrations {
		if _, err := d.conn.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

// QueryRow is a thin wrapper around sql.DB.QueryRow
func (d *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	return d.conn.QueryRow(query, args...)
}

// Query is a thin wrapper around sql.DB.Query
func (d *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return d.conn.Query(query, args...)
}

// Exec is a thin wrapper around sql.DB.Exec
func (d *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return d.conn.Exec(query, args...)
}

// Transaction executes a function within a database transaction
func (d *DB) Transaction(fn func(*sql.Tx) error) error {
	tx, err := d.conn.Begin()
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// Now returns the current time formatted for SQLite
func Now() string {
	return time.Now().Format("2006-01-02 15:04:05")
}
