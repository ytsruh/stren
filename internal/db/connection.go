// Package db provides a wrapper around Turso database operations.
// All external database dependencies are isolated here for easy replacement.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/pressly/goose/v3"
	turso "turso.tech/database/tursogo"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// resolveDBPath resolves the database path. If the path is relative (does not
// start with "/"), it is resolved relative to the current working directory.
// Absolute paths and the special ":memory:" string are returned unchanged.
func resolveDBPath(dbPath string) (string, error) {
	// In-memory databases and absolute paths need no resolution
	if dbPath == ":memory:" || filepath.IsAbs(dbPath) {
		return dbPath, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}

	return filepath.Join(cwd, dbPath), nil
}

// syncInterval defines how often the local database syncs with Turso Cloud.
// This value is hardcoded and not configurable via environment variables.
const syncInterval = 30 * time.Second

// DB wraps the sql.DB and turso.TursoSyncDb to provide application-specific operations.
type DB struct {
	conn    *sql.DB
	syncDb  *turso.TursoSyncDb
	closeCh chan struct{}
}

// NewConnection creates a new database connection with migrations applied.
// It ensures the parent directory of dbPath exists, creates a Turso Sync database,
// runs embedded migrations, pulls the latest remote changes, and starts a
// background sync goroutine.
func NewConnection(dbPath, tursoURL, tursoAuthToken string) (*DB, error) {
	// Resolve relative paths against the current working directory so that
	// DB_PATH=data/strength_tracker.db lands in the working directory
	// (e.g. /app in the Docker container).
	resolvedPath, err := resolveDBPath(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve database path: %w", err)
	}

	// Ensure the directory for the database file exists.
	// This is critical in production where /data is a mounted volume.
	if err := ensureDatabaseDir(resolvedPath); err != nil {
		return nil, fmt.Errorf("failed to initialize database directory: %w", err)
	}

	ctx := context.Background()

	syncDb, err := turso.NewTursoSyncDb(ctx, turso.TursoSyncDbConfig{
		Path:      resolvedPath,
		RemoteUrl: tursoURL,
		AuthToken: tursoAuthToken,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create turso sync db: %w", err)
	}

	conn, err := syncDb.Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := conn.Ping(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{
		conn:    conn,
		syncDb:  syncDb,
		closeCh: make(chan struct{}),
	}

	if err := db.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Pull the latest remote changes on startup.
	// This is non-fatal; the application can operate with local data if offline.
	if _, err := db.syncDb.Pull(ctx); err != nil {
		fmt.Printf("warning: failed to pull remote changes on startup: %v\n", err)
	}

	db.startBackgroundSync()

	return db, nil
}

// NewLocalConnection creates a new local-only Turso database without cloud sync.
// This is intended for testing and local development scenarios where
// Turso Cloud credentials are not available.
func NewLocalConnection(dbPath string) (*DB, error) {
	resolvedPath, err := resolveDBPath(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve database path: %w", err)
	}

	if err := ensureDatabaseDir(resolvedPath); err != nil {
		return nil, fmt.Errorf("failed to initialize database directory: %w", err)
	}

	conn, err := sql.Open("turso", resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open local database: %w", err)
	}

	if err := conn.Ping(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to ping local database: %w", err)
	}

	db := &DB{
		conn:    conn,
		closeCh: make(chan struct{}),
	}

	if err := db.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

// Conn returns the underlying *sql.DB connection.
// This is used by sqlc to create query objects.
func (d *DB) Conn() *sql.DB {
	return d.conn
}

// Close closes the database connection and stops the background sync goroutine.
func (d *DB) Close() error {
	if d.closeCh != nil {
		close(d.closeCh)
	}
	return d.conn.Close()
}

// ensureDatabaseDir verifies that the parent directory for the given database path
// exists and is a directory. If the directory does not exist, it attempts to create
// it. Returns clear, actionable errors for permission issues or path conflicts.
func ensureDatabaseDir(dbPath string) error {
	dir := filepath.Dir(dbPath)

	info, err := os.Stat(dir)
	if err == nil {
		if info.IsDir() {
			return nil
		}
		return fmt.Errorf("database path parent %q exists but is not a directory", dir)
	}

	if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check database directory %q: %w", dir, err)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("database directory %q does not exist and could not be created: %w", dir, err)
	}
	return nil
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

// startBackgroundSync starts a goroutine that periodically pushes local changes
// to Turso Cloud and pulls remote changes to the local database.
func (d *DB) startBackgroundSync() {
	go func() {
		ticker := time.NewTicker(syncInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				if err := d.syncDb.Push(ctx); err != nil {
					fmt.Printf("warning: background sync push failed: %v\n", err)
				}
				if _, err := d.syncDb.Pull(ctx); err != nil {
					fmt.Printf("warning: background sync pull failed: %v\n", err)
				}
				cancel()
			case <-d.closeCh:
				return
			}
		}
	}()
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
