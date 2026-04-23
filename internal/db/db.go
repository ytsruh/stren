// Package db provides a wrapper around SQLite database operations.
// All external database dependencies are isolated here for easy replacement.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"time"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB wraps the sql.DB to provide application-specific operations
type DB struct {
	conn *sql.DB
}

// New creates a new database connection with migrations applied.
// Migrations are embedded in the binary and executed automatically on startup.
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

// migrate runs embedded goose migrations against the database.
// It uses an embed.FS so that migrations are included in the compiled binary.
func (d *DB) migrate() error {
	fsys, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to open embedded migrations: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectSQLite3, d.conn, fsys)
	if err != nil {
		return fmt.Errorf("failed to create migration provider: %w", err)
	}

	ctx := context.Background()
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}
	fmt.Println("-------------------------------")
	fmt.Println("successfully applied migrations")
	fmt.Println("-------------------------------")
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
