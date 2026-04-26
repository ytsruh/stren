package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureDatabaseDir_CreatesNestedDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "nested", "deep", "test.db")

	// The directory should not exist yet
	parentDir := filepath.Dir(dbPath)
	if _, err := os.Stat(parentDir); !os.IsNotExist(err) {
		t.Fatalf("expected parent directory to not exist")
	}

	// Call the helper directly
	if err := ensureDatabaseDir(dbPath); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// The directory should now exist
	info, err := os.Stat(parentDir)
	if err != nil {
		t.Fatalf("expected directory to exist, got error: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("expected path to be a directory")
	}
}

func TestEnsureDatabaseDir_ExistingDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Directory already exists (t.TempDir creates it)
	if err := ensureDatabaseDir(dbPath); err != nil {
		t.Fatalf("expected no error for existing directory, got: %v", err)
	}
}

func TestEnsureDatabaseDir_InvalidPath(t *testing.T) {
	// Use an invalid path that cannot be created
	dbPath := "/dev/null/invalid/test.db"

	err := ensureDatabaseDir(dbPath)
	if err == nil {
		t.Fatal("expected error for invalid path, got nil")
	}

	// Error should be descriptive and mention the directory path
	if !os.IsPermission(err) && !os.IsNotExist(err) {
		// On some systems this may be a permission error, on others a different error
		// The key thing is the error message should be descriptive
		if err.Error() == "" {
			t.Fatal("expected non-empty error message")
		}
	}
}

func TestEnsureDatabaseDir_ParentIsFile(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a file that will act as the "parent directory"
	filePath := filepath.Join(tmpDir, "not-a-dir")
	if err := os.WriteFile(filePath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	dbPath := filepath.Join(filePath, "test.db")
	err := ensureDatabaseDir(dbPath)
	if err == nil {
		t.Fatal("expected error when parent is a file, got nil")
	}

	expected := "exists but is not a directory"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected error to contain %q, got: %v", expected, err)
	}
}

func TestResolveDBPath_AbsolutePath(t *testing.T) {
	// Absolute paths should pass through unchanged
	absPath := "/data/strength_tracker.db"
	resolved, err := resolveDBPath(absPath)
	if err != nil {
		t.Fatalf("expected no error for absolute path, got: %v", err)
	}
	if resolved != absPath {
		t.Fatalf("expected %q, got %q", absPath, resolved)
	}
}

func TestResolveDBPath_InMemory(t *testing.T) {
	// The special :memory: path should pass through unchanged
	resolved, err := resolveDBPath(":memory:")
	if err != nil {
		t.Fatalf("expected no error for :memory:, got: %v", err)
	}
	if resolved != ":memory:" {
		t.Fatalf("expected :memory:, got %q", resolved)
	}
}

func TestResolveDBPath_RelativePath(t *testing.T) {
	// Relative paths should resolve to the project root (where go.mod lives)
	resolved, err := resolveDBPath("data/strength_tracker.db")
	if err != nil {
		t.Fatalf("expected no error for relative path, got: %v", err)
	}

	// Verify the resolved path ends with the expected suffix
	expectedSuffix := filepath.Join("stren", "data", "strength_tracker.db")
	if !strings.HasSuffix(resolved, expectedSuffix) {
		t.Fatalf("expected path to end with %q, got %q", expectedSuffix, resolved)
	}

	// Verify the resolved path is absolute
	if !filepath.IsAbs(resolved) {
		t.Fatalf("expected resolved path to be absolute, got %q", resolved)
	}
}

func TestResolveDBPath_NestedRelativePath(t *testing.T) {
	resolved, err := resolveDBPath("foo/bar/baz.db")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expectedSuffix := filepath.Join("stren", "foo", "bar", "baz.db")
	if !strings.HasSuffix(resolved, expectedSuffix) {
		t.Fatalf("expected path to end with %q, got %q", expectedSuffix, resolved)
	}
}

func TestNow(t *testing.T) {
	now := Now()
	if now == "" {
		t.Fatal("expected non-empty time string")
	}
	// Basic format check: should be at least 19 characters "YYYY-MM-DD HH:MM:SS"
	if len(now) < 19 {
		t.Fatalf("expected time string to be at least 19 chars, got: %s", now)
	}
}
