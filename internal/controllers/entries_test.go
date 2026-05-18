package controllers

import (
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"stren/internal/models"
)

type mockRepository struct {
	mu        sync.Mutex
	exercises []models.Exercise
	entries   []models.ExerciseEntry

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
	errGetExerciseByID       error
}

func newMockRepository() *mockRepository {
	return &mockRepository{}
}

func (m *mockRepository) Create(_ *sql.Tx, name string) (string, error) {
	if m.errCreate != nil {
		return "", m.errCreate
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.exercises {
		if e.Name == name {
			return e.ID, nil
		}
	}
	id := "ex-" + name
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

	var exerciseID string
	for _, e := range m.exercises {
		if e.Name == entry.ExerciseName {
			exerciseID = e.ID
			break
		}
	}
	if exerciseID == "" {
		exerciseID = "ex-" + entry.ExerciseName
		m.exercises = append(m.exercises, models.Exercise{ID: exerciseID, Name: entry.ExerciseName})
	}
	entry.ExerciseID = exerciseID
	entry.ID = "entry-" + entry.ExerciseName
	m.entries = append(m.entries, *entry)
	return nil
}

func (m *mockRepository) GetEntry(id string, userID string) (*models.ExerciseEntry, error) {
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

func (m *mockRepository) UpdateEntry(entry *models.ExerciseEntry, userID string) error {
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

func (m *mockRepository) UpdateEntryWithDate(entry *models.ExerciseEntry, userID string) error {
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

func (m *mockRepository) DeleteEntry(id string, userID string) error {
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

func (m *mockRepository) ListEntries(userID string, limit int) ([]models.ExerciseEntry, error) {
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

func (m *mockRepository) GetEntriesByExercise(exerciseName string, userID string) ([]models.ExerciseEntry, error) {
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

func (m *mockRepository) GetEntriesByDateRange(start, end time.Time, userID string) ([]models.ExerciseEntry, error) {
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

func (m *mockRepository) ListEntriesLast30Days(userID string) ([]models.ExerciseEntry, error) {
	if m.errListEntries != nil {
		return nil, m.errListEntries
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	var result []models.ExerciseEntry
	for _, e := range m.entries {
		if e.CreatedAt.After(thirtyDaysAgo) && e.UserID == userID {
			result = append(result, e)
		}
	}
	return result, nil
}

func (m *mockRepository) GetExerciseByID(id string, userID string) (*models.Exercise, error) {
	if m.errGetExerciseByID != nil {
		return nil, m.errGetExerciseByID
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.exercises {
		if e.ID == id {
			cp := e
			return &cp, nil
		}
	}
	return nil, nil
}

func setupEntryController(t *testing.T) (*EntryController, *mockRepository) {
	t.Helper()
	mock := newMockRepository()
	mock.exercises = []models.Exercise{
		{ID: "ex-1", Name: "Squat"},
		{ID: "ex-2", Name: "Bench Press"},
	}
	return NewEntryController(mock), mock
}

func TestEntryController_ListEntries(t *testing.T) {
	ec, _ := setupEntryController(t)
	mock := ec.repo.(*mockRepository)
	mock.entries = []models.ExerciseEntry{
		{ID: "entry-1", UserID: "user-1", ExerciseName: "Squat", Reps: 5, Weight: 100},
		{ID: "entry-2", UserID: "user-1", ExerciseName: "Bench Press", Reps: 8, Weight: 80},
	}

	entries, err := ec.ListEntries("user-1")
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
		{ID: "entry-1", UserID: "user-1", ExerciseName: "Squat", Reps: 5, Weight: 100},
		{ID: "entry-2", UserID: "user-1", ExerciseName: "Bench Press", Reps: 8, Weight: 80},
	}

	entries, err := ec.ListEntries("user-1")
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
		{ID: "entry-1", UserID: "user-1", ExerciseName: "Squat", Reps: 5, Weight: 100},
	}

	entry, err := ec.GetEntry("entry-1", "user-1")
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
		{ID: "entry-1", UserID: "user-1", ExerciseName: "Squat", Reps: 5, Weight: 100},
	}

	entry, err := ec.GetEntry("entry-1", "user-999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry != nil {
		t.Error("expected nil entry for wrong user")
	}
}

func TestEntryController_CreateEntry(t *testing.T) {
	ec, _ := setupEntryController(t)

	entry, err := ec.CreateEntry("user-1", "Squat", "great set", 5, 100)
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

	_, err := ec.CreateEntry("user-1", "Squat", "great set", 5, 100)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestEntryController_UpdateEntry(t *testing.T) {
	ec, _ := setupEntryController(t)
	mock := ec.repo.(*mockRepository)
	mock.entries = []models.ExerciseEntry{
		{ID: "entry-1", UserID: "user-1", ExerciseName: "Squat", Reps: 5, Weight: 100},
	}

	entry, err := ec.UpdateEntry("entry-1", "user-1", "Squat", "even better", 6, 110, time.Now())
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
		{ID: "entry-1", UserID: "user-1", ExerciseName: "Squat", Reps: 5, Weight: 100},
	}

	err := ec.DeleteEntry("entry-1", "user-1")
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
		{ID: "entry-1", UserID: "user-1", ExerciseName: "Squat", Reps: 5, Weight: 100},
		{ID: "entry-2", UserID: "user-1", ExerciseName: "Bench Press", Reps: 8, Weight: 80},
		{ID: "entry-3", UserID: "user-1", ExerciseName: "Squat", Reps: 3, Weight: 110},
	}

	entries, err := ec.GetEntriesByExercise("Squat", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 squat entries, got %d", len(entries))
	}
}