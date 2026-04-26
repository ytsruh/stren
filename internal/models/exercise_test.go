package models

import (
	"database/sql"
	"testing"
	"time"

	"stren/internal/db"
)

// setupTestRepo creates a fresh in-memory database, a test user, and repositories for testing.
func setupTestRepo(t *testing.T) (*ExerciseRepository, *UserRepository, *db.DB, int64) {
	t.Helper()

	database, err := db.NewLocalConnection(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory database: %v", err)
	}

	userRepo := NewUserRepository(database)
	exerciseRepo := NewExerciseRepository(database)

	// Create a test user to associate with entries
	user := &User{
		Name:         "Test User",
		Email:        "test@example.com",
		PasswordHash: "hash",
	}
	if err := userRepo.CreateUser(user); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	return exerciseRepo, userRepo, database, user.ID
}

func TestExerciseRepository_CreateType(t *testing.T) {
	repo, _, database, _ := setupTestRepo(t)
	defer database.Close()

	id, err := repo.CreateType(nil, "Test Create Type")
	if err != nil {
		t.Fatalf("CreateType failed: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero id")
	}

	// Creating the same type again should return the existing ID.
	id2, err := repo.CreateType(nil, "Test Create Type")
	if err != nil {
		t.Fatalf("CreateType duplicate failed: %v", err)
	}
	if id2 != id {
		t.Fatalf("expected same id %d, got %d", id, id2)
	}
}

func TestExerciseRepository_CreateType_WithTx(t *testing.T) {
	repo, _, database, _ := setupTestRepo(t)
	defer database.Close()

	var createdID int64
	err := database.Transaction(func(tx *sql.Tx) error {
		id, err := repo.CreateType(tx, "Tx Type")
		createdID = id
		return err
	})
	if err != nil {
		t.Fatalf("transaction failed: %v", err)
	}
	if createdID == 0 {
		t.Fatal("expected non-zero id from transaction")
	}
}

func TestExerciseRepository_GetTypeByName(t *testing.T) {
	repo, _, database, _ := setupTestRepo(t)
	defer database.Close()

	// Seeded type should exist.
	et, err := repo.GetTypeByName("Squat")
	if err != nil {
		t.Fatalf("GetTypeByName failed: %v", err)
	}
	if et == nil {
		t.Fatal("expected seeded type 'Squat', got nil")
	}
	if et.Name != "Squat" {
		t.Fatalf("expected name 'Squat', got %q", et.Name)
	}

	// Non-existing type should return nil.
	et, err = repo.GetTypeByName("NonExistentExercise")
	if err != nil {
		t.Fatalf("GetTypeByName failed: %v", err)
	}
	if et != nil {
		t.Fatalf("expected nil for non-existing type, got %+v", et)
	}
}

func TestExerciseRepository_ListTypes(t *testing.T) {
	repo, _, database, _ := setupTestRepo(t)
	defer database.Close()

	types, err := repo.ListTypes()
	if err != nil {
		t.Fatalf("ListTypes failed: %v", err)
	}
	if len(types) != 10 {
		t.Fatalf("expected 10 seeded types, got %d", len(types))
	}

	// Verify ordering by name.
	if types[0].Name != "Barbell Row" {
		t.Fatalf("expected first type to be 'Barbell Row', got %q", types[0].Name)
	}
}

func TestExerciseRepository_CreateEntry(t *testing.T) {
	repo, _, database, userID := setupTestRepo(t)
	defer database.Close()

	entry := &ExerciseEntry{
		ExerciseName: "Squat",
		UserID:       userID,
		Reps:         5,
		Weight:       140.0,
		Notes:        "Test entry",
		CreatedAt:    time.Now(),
	}

	err := repo.CreateEntry(entry)
	if err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}
	if entry.ID == 0 {
		t.Fatal("expected non-zero entry ID after creation")
	}
}

func TestExerciseRepository_GetEntry(t *testing.T) {
	repo, _, database, userID := setupTestRepo(t)
	defer database.Close()

	// Non-existing entry.
	entry, err := repo.GetEntry(9999, userID)
	if err != nil {
		t.Fatalf("GetEntry failed: %v", err)
	}
	if entry != nil {
		t.Fatalf("expected nil for non-existing entry, got %+v", entry)
	}

	// Create an entry and retrieve it.
	created := &ExerciseEntry{
		ExerciseName: "Bench Press",
		UserID:       userID,
		Reps:         8,
		Weight:       80.0,
		Notes:        "",
		CreatedAt:    time.Now(),
	}
	if err := repo.CreateEntry(created); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}

	entry, err = repo.GetEntry(created.ID, userID)
	if err != nil {
		t.Fatalf("GetEntry failed: %v", err)
	}
	if entry == nil {
		t.Fatal("expected entry, got nil")
	}
	if entry.ExerciseName != "Bench Press" {
		t.Fatalf("expected exercise name 'Bench Press', got %q", entry.ExerciseName)
	}
	if entry.Reps != 8 {
		t.Fatalf("expected reps 8, got %d", entry.Reps)
	}
	if entry.Weight != 80.0 {
		t.Fatalf("expected weight 80.0, got %f", entry.Weight)
	}
}

func TestExerciseRepository_UpdateEntry(t *testing.T) {
	repo, _, database, userID := setupTestRepo(t)
	defer database.Close()

	created := &ExerciseEntry{
		ExerciseName: "Deadlift",
		UserID:       userID,
		Reps:         5,
		Weight:       180.0,
		Notes:        "heavy",
		CreatedAt:    time.Now(),
	}
	if err := repo.CreateEntry(created); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}

	created.Reps = 3
	created.Weight = 200.0
	created.Notes = "pr"
	created.ExerciseName = "Deadlift"

	if err := repo.UpdateEntry(created, userID); err != nil {
		t.Fatalf("UpdateEntry failed: %v", err)
	}

	updated, err := repo.GetEntry(created.ID, userID)
	if err != nil {
		t.Fatalf("GetEntry failed: %v", err)
	}
	if updated.Reps != 3 {
		t.Fatalf("expected reps 3, got %d", updated.Reps)
	}
	if updated.Weight != 200.0 {
		t.Fatalf("expected weight 200.0, got %f", updated.Weight)
	}
	if updated.Notes != "pr" {
		t.Fatalf("expected notes 'pr', got %q", updated.Notes)
	}
}

func TestExerciseRepository_UpdateEntryWithDate(t *testing.T) {
	repo, _, database, userID := setupTestRepo(t)
	defer database.Close()

	originalTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	created := &ExerciseEntry{
		ExerciseName: "Overhead Press",
		UserID:       userID,
		Reps:         5,
		Weight:       60.0,
		Notes:        "",
		CreatedAt:    originalTime,
	}
	if err := repo.CreateEntry(created); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}

	newTime := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)
	created.Reps = 6
	created.CreatedAt = newTime

	if err := repo.UpdateEntryWithDate(created, userID); err != nil {
		t.Fatalf("UpdateEntryWithDate failed: %v", err)
	}

	updated, err := repo.GetEntry(created.ID, userID)
	if err != nil {
		t.Fatalf("GetEntry failed: %v", err)
	}
	if updated.Reps != 6 {
		t.Fatalf("expected reps 6, got %d", updated.Reps)
	}
	if !updated.CreatedAt.Equal(newTime) {
		t.Fatalf("expected created_at %v, got %v", newTime, updated.CreatedAt)
	}
}

func TestExerciseRepository_DeleteEntry(t *testing.T) {
	repo, _, database, userID := setupTestRepo(t)
	defer database.Close()

	created := &ExerciseEntry{
		ExerciseName: "Pull Up",
		UserID:       userID,
		Reps:         10,
		Weight:       0,
		Notes:        "bw",
		CreatedAt:    time.Now(),
	}
	if err := repo.CreateEntry(created); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}

	if err := repo.DeleteEntry(created.ID, userID); err != nil {
		t.Fatalf("DeleteEntry failed: %v", err)
	}

	entry, err := repo.GetEntry(created.ID, userID)
	if err != nil {
		t.Fatalf("GetEntry failed: %v", err)
	}
	if entry != nil {
		t.Fatalf("expected nil after delete, got %+v", entry)
	}
}

func TestExerciseRepository_ListEntries(t *testing.T) {
	repo, _, database, userID := setupTestRepo(t)
	defer database.Close()

	// Create multiple entries.
	for i := 0; i < 5; i++ {
		entry := &ExerciseEntry{
			ExerciseName: "Dips",
			UserID:       userID,
			Reps:         i + 1,
			Weight:       float64(i * 10),
			CreatedAt:    time.Now().Add(time.Duration(i) * time.Hour),
		}
		if err := repo.CreateEntry(entry); err != nil {
			t.Fatalf("CreateEntry failed: %v", err)
		}
	}

	// With limit.
	entries, err := repo.ListEntries(userID, 3)
	if err != nil {
		t.Fatalf("ListEntries failed: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries with limit, got %d", len(entries))
	}

	// Without limit.
	entries, err = repo.ListEntries(userID, 0)
	if err != nil {
		t.Fatalf("ListEntries failed: %v", err)
	}
	if len(entries) != 5 {
		t.Fatalf("expected 5 entries without limit, got %d", len(entries))
	}

	// Verify descending order (latest first).
	if entries[0].CreatedAt.Before(entries[1].CreatedAt) {
		t.Fatal("expected entries in descending order by created_at")
	}
}

func TestExerciseRepository_GetEntriesByExercise(t *testing.T) {
	repo, _, database, userID := setupTestRepo(t)
	defer database.Close()

	// Create entries for different exercises.
	if err := repo.CreateEntry(&ExerciseEntry{ExerciseName: "Squat", UserID: userID, Reps: 5, Weight: 100, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}
	if err := repo.CreateEntry(&ExerciseEntry{ExerciseName: "Bench Press", UserID: userID, Reps: 8, Weight: 80, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}
	if err := repo.CreateEntry(&ExerciseEntry{ExerciseName: "Squat", UserID: userID, Reps: 5, Weight: 105, CreatedAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}

	entries, err := repo.GetEntriesByExercise("Squat", userID)
	if err != nil {
		t.Fatalf("GetEntriesByExercise failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 squat entries, got %d", len(entries))
	}

	for _, e := range entries {
		if e.ExerciseName != "Squat" {
			t.Fatalf("expected exercise name 'Squat', got %q", e.ExerciseName)
		}
	}
}

func TestExerciseRepository_GetEntriesByDateRange(t *testing.T) {
	repo, _, database, userID := setupTestRepo(t)
	defer database.Close()

	loc := time.UTC
	day1 := time.Date(2024, 1, 1, 10, 0, 0, 0, loc)
	day2 := time.Date(2024, 1, 2, 10, 0, 0, 0, loc)
	day3 := time.Date(2024, 1, 3, 10, 0, 0, 0, loc)

	if err := repo.CreateEntry(&ExerciseEntry{ExerciseName: "Squat", UserID: userID, Reps: 5, Weight: 100, CreatedAt: day1}); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}
	if err := repo.CreateEntry(&ExerciseEntry{ExerciseName: "Squat", UserID: userID, Reps: 5, Weight: 105, CreatedAt: day2}); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}
	if err := repo.CreateEntry(&ExerciseEntry{ExerciseName: "Squat", UserID: userID, Reps: 5, Weight: 110, CreatedAt: day3}); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}

	entries, err := repo.GetEntriesByDateRange(day1, day2, userID)
	if err != nil {
		t.Fatalf("GetEntriesByDateRange failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries in range, got %d", len(entries))
	}
}
