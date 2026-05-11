package controllers

import (
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"stren/internal/models"
)

// mockRepository is an in-memory implementation of models.Repository for testing.
type mockRepository struct {
	mu        sync.Mutex
	exercises []models.Exercise
	entries   []models.ExerciseEntry

	nextExerciseID int64
	nextEntryID    int64

	errCreate            error
	errGetByName          error
	errList               error
	errCreateEntry        error
	errGetEntry           error
	errUpdateEntry        error
	errUpdateEntryWithDate error
	errDeleteEntry        error
	errListEntries        error
	errGetEntriesByExercise error
	errGetEntriesByDateRange error
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		nextExerciseID: 1,
		nextEntryID:    1,
	}
}

func (m *mockRepository) Create(_ *sql.Tx, name string) (int64, error) {
	if m.errCreate != nil {
		return 0, m.errCreate
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.exercises {
		if e.Name == name {
			return e.ID, nil
		}
	}
	id := m.nextExerciseID
	m.nextExerciseID++
	m.exercises = append(m.exercises, models.Exercise{ID: id, Name: name})
	return id, nil
}

func (m *mockRepository) GetByName(name string) (*models.Exercise, error) {
	if m.errGetByName != nil {
		return nil, m.errGetByName
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.exercises {
		if e.Name == name {
			cp := e
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockRepository) List() ([]models.Exercise, error) {
	if m.errList != nil {
		return nil, m.errList
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]models.Exercise, len(m.exercises))
	copy(result, m.exercises)
	return result, nil
}

func (m *mockRepository) CreateEntry(entry *models.ExerciseEntry) error {
	if m.errCreateEntry != nil {
		return m.errCreateEntry
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var exerciseID int64
	for _, e := range m.exercises {
		if e.Name == entry.ExerciseName {
			exerciseID = e.ID
			break
		}
	}
	if exerciseID == 0 {
		exerciseID = m.nextExerciseID
		m.nextExerciseID++
		m.exercises = append(m.exercises, models.Exercise{ID: exerciseID, Name: entry.ExerciseName})
	}
	entry.ExerciseID = exerciseID
	entry.ID = m.nextEntryID
	m.nextEntryID++
	m.entries = append(m.entries, *entry)
	return nil
}

func (m *mockRepository) GetEntry(id int64, userID int64) (*models.ExerciseEntry, error) {
	if m.errGetEntry != nil {
		return nil, m.errGetEntry
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.entries {
		if e.ID == id && e.UserID == userID {
			cp := e
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockRepository) UpdateEntry(entry *models.ExerciseEntry, userID int64) error {
	if m.errUpdateEntry != nil {
		return m.errUpdateEntry
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, e := range m.entries {
		if e.ID == entry.ID && e.UserID == userID {
			m.entries[i] = *entry
			return nil
		}
	}
	return nil
}

func (m *mockRepository) UpdateEntryWithDate(entry *models.ExerciseEntry, userID int64) error {
	if m.errUpdateEntryWithDate != nil {
		return m.errUpdateEntryWithDate
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, e := range m.entries {
		if e.ID == entry.ID && e.UserID == userID {
			m.entries[i] = *entry
			return nil
		}
	}
	return nil
}

func (m *mockRepository) DeleteEntry(id int64, userID int64) error {
	if m.errDeleteEntry != nil {
		return m.errDeleteEntry
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, e := range m.entries {
		if e.ID == id && e.UserID == userID {
			m.entries = append(m.entries[:i], m.entries[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockRepository) ListEntries(userID int64, limit int) ([]models.ExerciseEntry, error) {
	if m.errListEntries != nil {
		return nil, m.errListEntries
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []models.ExerciseEntry
	for _, e := range m.entries {
		if e.UserID == userID {
			result = append(result, e)
		}
	}
	if limit > 0 && limit < len(result) {
		result = result[:limit]
	}
	return result, nil
}

func (m *mockRepository) GetEntriesByExercise(exerciseName string, userID int64) ([]models.ExerciseEntry, error) {
	if m.errGetEntriesByExercise != nil {
		return nil, m.errGetEntriesByExercise
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []models.ExerciseEntry
	for _, e := range m.entries {
		if e.ExerciseName == exerciseName && e.UserID == userID {
			result = append(result, e)
		}
	}
	return result, nil
}

func (m *mockRepository) GetEntriesByDateRange(start, end time.Time, userID int64) ([]models.ExerciseEntry, error) {
	if m.errGetEntriesByDateRange != nil {
		return nil, m.errGetEntriesByDateRange
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []models.ExerciseEntry
	for _, e := range m.entries {
		if !e.CreatedAt.Before(start) && !e.CreatedAt.After(end) && e.UserID == userID {
			result = append(result, e)
		}
	}
	return result, nil
}



func setupEntryController(t *testing.T) (*EntryController, *mockRepository) {
	t.Helper()
	mock := newMockRepository()
	mock.exercises = []models.Exercise{
		{ID: 1, Name: "Squat"},
		{ID: 2, Name: "Bench Press"},
	}
	mock.nextExerciseID = 3
	return NewEntryController(mock), mock
}

func TestEntryController_ListEntries(t *testing.T) {
	ec, _ := setupEntryController(t)
	mock := ec.repo.(*mockRepository)
	mock.entries = []models.ExerciseEntry{
		{ID: 1, UserID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100},
		{ID: 2, UserID: 1, ExerciseName: "Bench Press", Reps: 8, Weight: 80},
	}

	entries, err := ec.ListEntries(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}

func TestEntryController_ListEntries_Limit(t *testing.T) {
	ec, mock := setupEntryController(t)
	mock.entries = []models.ExerciseEntry{
		{ID: 1, UserID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100},
		{ID: 2, UserID: 1, ExerciseName: "Bench Press", Reps: 8, Weight: 80},
	}

	entries, err := ec.ListEntries(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}

func TestEntryController_GetEntry(t *testing.T) {
	ec, _ := setupEntryController(t)
	mock := ec.repo.(*mockRepository)
	mock.entries = []models.ExerciseEntry{
		{ID: 1, UserID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100},
	}

	entry, err := ec.GetEntry(1, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.ExerciseName != "Squat" {
		t.Errorf("expected 'Squat', got %q", entry.ExerciseName)
	}
}

func TestEntryController_GetEntry_WrongUser(t *testing.T) {
	ec, _ := setupEntryController(t)
	mock := ec.repo.(*mockRepository)
	mock.entries = []models.ExerciseEntry{
		{ID: 1, UserID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100},
	}

	entry, err := ec.GetEntry(1, 999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry != nil {
		t.Error("expected nil entry for wrong user")
	}
}

func TestEntryController_CreateEntry(t *testing.T) {
	ec, _ := setupEntryController(t)

	entry, err := ec.CreateEntry(1, "Squat", "great set", 5, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.ExerciseName != "Squat" {
		t.Errorf("expected 'Squat', got %q", entry.ExerciseName)
	}
	if entry.Reps != 5 {
		t.Errorf("expected 5 reps, got %d", entry.Reps)
	}
}

func TestEntryController_CreateEntry_RepositoryError(t *testing.T) {
	ec, mock := setupEntryController(t)
	mock.errCreateEntry = errors.New("db error")

	_, err := ec.CreateEntry(1, "Squat", "great set", 5, 100)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestEntryController_UpdateEntry(t *testing.T) {
	ec, _ := setupEntryController(t)
	mock := ec.repo.(*mockRepository)
	mock.entries = []models.ExerciseEntry{
		{ID: 1, UserID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100},
	}

	entry, err := ec.UpdateEntry(1, 1, "Squat", "even better", 6, 110, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.Reps != 6 {
		t.Errorf("expected 6 reps, got %d", entry.Reps)
	}
}

func TestEntryController_DeleteEntry(t *testing.T) {
	ec, _ := setupEntryController(t)
	mock := ec.repo.(*mockRepository)
	mock.entries = []models.ExerciseEntry{
		{ID: 1, UserID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100},
	}

	err := ec.DeleteEntry(1, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.entries) != 0 {
		t.Fatalf("expected entry to be deleted, got %d entries", len(mock.entries))
	}
}

func TestEntryController_List(t *testing.T) {
	ec, _ := setupEntryController(t)

	exercises, err := ec.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(exercises) != 2 {
		t.Fatalf("expected 2 exercises, got %d", len(exercises))
	}
}

func TestEntryController_List_Error(t *testing.T) {
	ec, mock := setupEntryController(t)
	mock.errList = errors.New("db error")

	_, err := ec.List()
	if err == nil {
		t.Fatal("expected error from repository")
	}
}

func TestEntryController_GetEntriesByExercise(t *testing.T) {
	ec, mock := setupEntryController(t)
	mock.entries = []models.ExerciseEntry{
		{ID: 1, UserID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100},
		{ID: 2, UserID: 1, ExerciseName: "Bench Press", Reps: 8, Weight: 80},
		{ID: 3, UserID: 1, ExerciseName: "Squat", Reps: 3, Weight: 110},
	}

	entries, err := ec.GetEntriesByExercise("Squat", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 squat entries, got %d", len(entries))
	}
}