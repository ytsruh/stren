package models

import (
	"database/sql"
	"testing"
	"time"

	"stren/internal/db"
)

func setupTestRepo(t *testing.T) (*ExerciseRepository, *UserRepository, *db.DB, string) {
	t.Helper()

	database, err := db.NewLocalConnection(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory database: %v", err)
	}

	userRepo := NewUserRepository(database)
	exerciseRepo := NewExerciseRepository(database)

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

func TestExerciseRepository_Create(t *testing.T) {
	repo, _, database, _ := setupTestRepo(t)
	defer database.Close()

	id, err := repo.Create(nil, "Test Create")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty id")
	}

	_, err = repo.Create(nil, "Test Create")
	if err == nil {
		t.Fatal("expected error for duplicate name, got nil")
	}
}

func TestExerciseRepository_Create_WithTx(t *testing.T) {
	repo, _, database, _ := setupTestRepo(t)
	defer database.Close()

	var createdID string
	err := database.Transaction(func(tx *sql.Tx) error {
		id, err := repo.Create(tx, "Tx Exercise")
		createdID = id
		return err
	})
	if err != nil {
		t.Fatalf("transaction failed: %v", err)
	}
	if createdID == "" {
		t.Fatal("expected non-empty id from transaction")
	}
}

func TestExerciseRepository_GetByName(t *testing.T) {
	repo, _, database, _ := setupTestRepo(t)
	defer database.Close()

	_, err := repo.Create(nil, "Test Exercise")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	et, err := repo.GetByName("Test Exercise")
	if err != nil {
		t.Fatalf("GetByName failed: %v", err)
	}
	if et == nil {
		t.Fatal("expected exercise, got nil")
	}
	if et.Name != "Test Exercise" {
		t.Fatalf("expected name 'Test Exercise', got %q", et.Name)
	}

	et, err = repo.GetByName("NonExistentExercise")
	if err != nil {
		t.Fatalf("GetByName failed: %v", err)
	}
	if et != nil {
		t.Fatalf("expected nil for non-existing exercise, got %+v", et)
	}
}

func TestExerciseRepository_List(t *testing.T) {
	repo, _, database, _ := setupTestRepo(t)
	defer database.Close()

	_, err := repo.Create(nil, "Alpha")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_, err = repo.Create(nil, "Beta")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	exercises, err := repo.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(exercises) < 2 {
		t.Fatalf("expected at least 2 exercises, got %d", len(exercises))
	}

	if exercises[0].Name > exercises[1].Name {
		t.Fatalf("expected exercises ordered by name, got %q then %q", exercises[0].Name, exercises[1].Name)
	}
}

func TestExerciseRepository_CreateEntry(t *testing.T) {
	repo, _, database, userID := setupTestRepo(t)
	defer database.Close()

	exercise, err := repo.Create(nil, "Squat")
	if err != nil {
		t.Fatalf("Create exercise failed: %v", err)
	}

	entry := &ExerciseEntry{
		ExerciseID: exercise,
		UserID:     userID,
		Reps:       5,
		Weight:     140.0,
		Notes:      "Test entry",
		CreatedAt:  time.Now(),
	}

	err = repo.CreateEntry(entry)
	if err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}
	if entry.ID == "" {
		t.Fatal("expected non-empty entry ID after creation")
	}
}

func TestExerciseRepository_GetEntry(t *testing.T) {
	repo, _, database, userID := setupTestRepo(t)
	defer database.Close()

	exercise, err := repo.Create(nil, "Bench Press")
	if err != nil {
		t.Fatalf("Create exercise failed: %v", err)
	}

	created := &ExerciseEntry{
		ExerciseID: exercise,
		UserID:     userID,
		Reps:       8,
		Weight:     80.0,
		Notes:      "",
		CreatedAt:  time.Now(),
	}
	if err := repo.CreateEntry(created); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}

	entry, err := repo.GetEntry(created.ID, userID)
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

	exercise, err := repo.Create(nil, "Deadlift")
	if err != nil {
		t.Fatalf("Create exercise failed: %v", err)
	}

	created := &ExerciseEntry{
		ExerciseID: exercise,
		UserID:     userID,
		Reps:       5,
		Weight:     180.0,
		Notes:      "heavy",
		CreatedAt:  time.Now(),
	}
	if err := repo.CreateEntry(created); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}

	created.Reps = 3
	created.Weight = 200.0
	created.Notes = "pr"

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

	exercise, err := repo.Create(nil, "Overhead Press")
	if err != nil {
		t.Fatalf("Create exercise failed: %v", err)
	}

	originalTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	created := &ExerciseEntry{
		ExerciseID: exercise,
		UserID:     userID,
		Reps:       5,
		Weight:     60.0,
		Notes:      "",
		CreatedAt:  originalTime,
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

	exercise, err := repo.Create(nil, "Pull Up")
	if err != nil {
		t.Fatalf("Create exercise failed: %v", err)
	}

	created := &ExerciseEntry{
		ExerciseID: exercise,
		UserID:     userID,
		Reps:       10,
		Weight:     0,
		Notes:      "bw",
		CreatedAt:  time.Now(),
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

	exercise, err := repo.Create(nil, "Dips")
	if err != nil {
		t.Fatalf("Create exercise failed: %v", err)
	}

	for i := 0; i < 5; i++ {
		entry := &ExerciseEntry{
			ExerciseID: exercise,
			UserID:     userID,
			Reps:       i + 1,
			Weight:     float64(i * 10),
			CreatedAt:  time.Now().Add(time.Duration(i) * time.Hour),
		}
		if err := repo.CreateEntry(entry); err != nil {
			t.Fatalf("CreateEntry failed: %v", err)
		}
	}

	entries, err := repo.ListEntries(userID, 3)
	if err != nil {
		t.Fatalf("ListEntries failed: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries with limit, got %d", len(entries))
	}

	entries, err = repo.ListEntries(userID, 0)
	if err != nil {
		t.Fatalf("ListEntries failed: %v", err)
	}
	if len(entries) != 5 {
		t.Fatalf("expected 5 entries without limit, got %d", len(entries))
	}

	if entries[0].CreatedAt.Before(entries[1].CreatedAt) {
		t.Fatal("expected entries in descending order by created_at")
	}
}

func TestExerciseRepository_GetEntriesByExercise(t *testing.T) {
	repo, _, database, userID := setupTestRepo(t)
	defer database.Close()

	squat, err := repo.Create(nil, "Squat")
	if err != nil {
		t.Fatalf("Create exercise failed: %v", err)
	}
	bench, err := repo.Create(nil, "Bench Press")
	if err != nil {
		t.Fatalf("Create exercise failed: %v", err)
	}

	if err := repo.CreateEntry(&ExerciseEntry{ExerciseID: squat, UserID: userID, Reps: 5, Weight: 100, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}
	if err := repo.CreateEntry(&ExerciseEntry{ExerciseID: bench, UserID: userID, Reps: 8, Weight: 80, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}
	if err := repo.CreateEntry(&ExerciseEntry{ExerciseID: squat, UserID: userID, Reps: 5, Weight: 105, CreatedAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}

	entries, err := repo.GetEntriesByExercise(squat, userID)
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
		if e.ExerciseID != squat {
			t.Fatalf("expected exercise ID %q, got %q", squat, e.ExerciseID)
		}
	}
}

func TestExerciseRepository_GetEntriesByDateRange(t *testing.T) {
	repo, _, database, userID := setupTestRepo(t)
	defer database.Close()

	exercise, err := repo.Create(nil, "Squat")
	if err != nil {
		t.Fatalf("Create exercise failed: %v", err)
	}

	loc := time.UTC
	day1 := time.Date(2024, 1, 1, 10, 0, 0, 0, loc)
	day2 := time.Date(2024, 1, 2, 10, 0, 0, 0, loc)
	day3 := time.Date(2024, 1, 3, 10, 0, 0, 0, loc)

	if err := repo.CreateEntry(&ExerciseEntry{ExerciseID: exercise, UserID: userID, Reps: 5, Weight: 100, CreatedAt: day1}); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}
	if err := repo.CreateEntry(&ExerciseEntry{ExerciseID: exercise, UserID: userID, Reps: 5, Weight: 105, CreatedAt: day2}); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}
	if err := repo.CreateEntry(&ExerciseEntry{ExerciseID: exercise, UserID: userID, Reps: 5, Weight: 110, CreatedAt: day3}); err != nil {
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