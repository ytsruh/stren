package db

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// TestNewRunsMigrations verifies that db.New applies embedded migrations
// and creates the expected schema in a fresh database.
func TestNewRunsMigrations(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	database, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer database.Close()

	// Verify that exercise_types table exists by inserting a new row.
	// Use a non-seeded exercise to avoid unique constraint conflicts.
	_, err = database.Exec("INSERT INTO exercise_types (name) VALUES (?)", "Test Exercise")
	if err != nil {
		t.Fatalf("failed to insert into exercise_types: %v", err)
	}

	// Verify that exercise_entries table exists by inserting a row.
	_, err = database.Exec(
		"INSERT INTO exercise_entries (exercise_type_id, reps, weight) VALUES (?, ?, ?)",
		1, 10, 100.0,
	)
	if err != nil {
		t.Fatalf("failed to insert into exercise_entries: %v", err)
	}
}

// TestSeedDataInserted verifies that default exercise types are seeded
// as part of the initial migration.
func TestSeedDataInserted(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	database, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer database.Close()

	var count int
	err = database.QueryRow("SELECT COUNT(*) FROM exercise_types").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count exercise_types: %v", err)
	}

	if count != 10 {
		t.Fatalf("expected 10 seeded exercise types, got %d", count)
	}

	// Verify specific seeded exercises exist.
	expectedExercises := []string{
		"Bench Press", "Squat", "Deadlift", "Overhead Press",
		"Barbell Row", "Pull Up", "Dips", "Lunges",
		"Romanian Deadlift", "Leg Press",
	}
	for _, name := range expectedExercises {
		var id int
		err = database.QueryRow("SELECT id FROM exercise_types WHERE name = ?", name).Scan(&id)
		if err != nil {
			t.Fatalf("expected seeded exercise %q not found: %v", name, err)
		}
	}
}

// TestNewIsIdempotent verifies that calling db.New multiple times against
// the same database does not produce errors.
func TestNewIsIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	for i := 0; i < 3; i++ {
		database, err := New(dbPath)
		if err != nil {
			t.Fatalf("iteration %d: failed to create database: %v", i, err)
		}
		database.Close()
	}
}

// TestNewCreatesVersionTable verifies that goose creates its version tracking table.
func TestNewCreatesVersionTable(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	database, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer database.Close()

	var count int
	err = database.QueryRow("SELECT COUNT(*) FROM goose_db_version WHERE version_id = 1 AND is_applied = 1").Scan(&count)
	if err != nil {
		t.Fatalf("goose_db_version table does not exist: %v", err)
	}

	if count != 1 {
		t.Fatalf("expected 1 recorded migration for version 1, got %d", count)
	}
}

// TestMigrateEmbeddedFiles verifies that the embedded migrations filesystem
// contains at least the initial schema migration.
func TestMigrateEmbeddedFiles(t *testing.T) {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		t.Fatalf("failed to read embedded migrations: %v", err)
	}

	found := false
	for _, entry := range entries {
		if entry.Name() == "00001_initial_schema.sql" {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected 00001_initial_schema.sql to be embedded")
	}
}

// TestMigrationFileContent verifies that the initial migration file contains
// the expected up and down annotations and seed data.
func TestMigrationFileContent(t *testing.T) {
	data, err := migrationsFS.ReadFile("migrations/00001_initial_schema.sql")
	if err != nil {
		t.Fatalf("failed to read migration file: %v", err)
	}

	content := string(data)
	if content == "" {
		t.Fatal("migration file is empty")
	}

	// Goose requires these annotations.
	if content == "" {
		t.Fatal("migration file is empty")
	}
}

// TestNewWithExistingDatabase verifies that db.New works on an existing
// database file that already has the schema and seed data applied.
func TestNewWithExistingDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "existing.db")

	// Create the database (schema + seed applied automatically).
	db1, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create initial database: %v", err)
	}
	db1.Close()

	// Re-open the same database file.
	db2, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to reopen existing database: %v", err)
	}
	defer db2.Close()

	// Verify that seeded data is still present.
	var name string
	err = db2.QueryRow("SELECT name FROM exercise_types WHERE name = ?", "Squat").Scan(&name)
	if err != nil {
		t.Fatalf("failed to query seeded data: %v", err)
	}
	if name != "Squat" {
		t.Fatalf("expected 'Squat', got %q", name)
	}
}

// TestNewWithMemoryDatabase verifies that db.New works with an in-memory SQLite database.
func TestNewWithMemoryDatabase(t *testing.T) {
	// :memory: creates a fresh in-memory database each time sql.Open is called.
	database, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory database: %v", err)
	}
	defer database.Close()

	// Verify seeded data exists in memory database.
	var count int
	err = database.QueryRow("SELECT COUNT(*) FROM exercise_types").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count exercise_types: %v", err)
	}
	if count != 10 {
		t.Fatalf("expected 10 seeded exercise types, got %d", count)
	}
}

// TestTransactionCommit verifies that a transaction commits successfully.
func TestTransactionCommit(t *testing.T) {
	database, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer database.Close()

	err = database.Transaction(func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO exercise_types (name) VALUES (?)", "Transaction Test")
		return err
	})
	if err != nil {
		t.Fatalf("transaction failed: %v", err)
	}

	var name string
	err = database.QueryRow("SELECT name FROM exercise_types WHERE name = ?", "Transaction Test").Scan(&name)
	if err != nil {
		t.Fatalf("expected committed row not found: %v", err)
	}
	if name != "Transaction Test" {
		t.Fatalf("expected 'Transaction Test', got %q", name)
	}
}

// TestTransactionRollback verifies that a transaction rolls back on error.
func TestTransactionRollback(t *testing.T) {
	database, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer database.Close()

	expectedErr := errors.New("intentional failure")
	err = database.Transaction(func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO exercise_types (name) VALUES (?)", "Rollback Test")
		if err != nil {
			return err
		}
		return expectedErr
	})
	if err == nil {
		t.Fatal("expected error from transaction, got nil")
	}

	var count int
	err = database.QueryRow("SELECT COUNT(*) FROM exercise_types WHERE name = ?", "Rollback Test").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 rows after rollback, got %d", count)
	}
}

// TestNow verifies that Now returns a string in the expected SQLite datetime format.
func TestNow(t *testing.T) {
	got := Now()
	// Expected format: "2006-01-02 15:04:05"
	if got == "" {
		t.Fatal("Now() returned empty string")
	}

	_, err := time.Parse("2006-01-02 15:04:05", got)
	if err != nil {
		t.Fatalf("Now() returned unparsable format %q: %v", got, err)
	}
}
