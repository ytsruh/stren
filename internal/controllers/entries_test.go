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
	mu      sync.Mutex
	types   []models.ExerciseType
	entries []models.ExerciseEntry

	nextTypeID  int64
	nextEntryID int64

	errCreateType            error
	errGetTypeByName         error
	errListTypes             error
	errCreateEntry           error
	errGetEntry              error
	errUpdateEntry           error
	errUpdateEntryWithDate   error
	errDeleteEntry           error
	errListEntries           error
	errGetEntriesByExercise  error
	errGetEntriesByDateRange error
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		nextTypeID:  1,
		nextEntryID: 1,
	}
}

func (m *mockRepository) CreateType(_ *sql.Tx, name string) (int64, error) {
	if m.errCreateType != nil {
		return 0, m.errCreateType
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.types {
		if t.Name == name {
			return t.ID, nil
		}
	}
	id := m.nextTypeID
	m.nextTypeID++
	m.types = append(m.types, models.ExerciseType{ID: id, Name: name})
	return id, nil
}

func (m *mockRepository) GetTypeByName(name string) (*models.ExerciseType, error) {
	if m.errGetTypeByName != nil {
		return nil, m.errGetTypeByName
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.types {
		if t.Name == name {
			cp := t
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockRepository) ListTypes() ([]models.ExerciseType, error) {
	if m.errListTypes != nil {
		return nil, m.errListTypes
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]models.ExerciseType, len(m.types))
	copy(result, m.types)
	return result, nil
}

func (m *mockRepository) CreateEntry(entry *models.ExerciseEntry) error {
	if m.errCreateEntry != nil {
		return m.errCreateEntry
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var typeID int64
	for _, t := range m.types {
		if t.Name == entry.ExerciseName {
			typeID = t.ID
			break
		}
	}
	if typeID == 0 {
		typeID = m.nextTypeID
		m.nextTypeID++
		m.types = append(m.types, models.ExerciseType{ID: typeID, Name: entry.ExerciseName})
	}
	entry.ExerciseTypeID = typeID
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
	mock.types = []models.ExerciseType{
		{ID: 1, Name: "Squat"},
		{ID: 2, Name: "Bench Press"},
	}
	mock.nextTypeID = 3
	return NewEntryController(mock), mock
}

func TestEntryController_ListEntries(t *testing.T) {
	ec, mock := setupEntryController(t)
	mock.entries = []models.ExerciseEntry{
		{ID: 1, UserID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100},
		{ID: 2, UserID: 1, ExerciseName: "Bench Press", Reps: 8, Weight: 80},
		{ID: 3, UserID: 2, ExerciseName: "Deadlift", Reps: 3, Weight: 180},
	}

	entries, err := ec.ListEntries(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries for user 1, got %d", len(entries))
	}
}

func TestEntryController_ListEntries_Limit(t *testing.T) {
	ec, mock := setupEntryController(t)
	for i := 0; i < 150; i++ {
		mock.entries = append(mock.entries, models.ExerciseEntry{
			ID: int64(i + 1), UserID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100,
		})
	}

	entries, err := ec.ListEntries(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 100 {
		t.Fatalf("expected 100 entries (limit), got %d", len(entries))
	}
}

func TestEntryController_GetEntry(t *testing.T) {
	ec, mock := setupEntryController(t)
	mock.entries = []models.ExerciseEntry{
		{ID: 1, UserID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100},
	}

	entry, err := ec.GetEntry(1, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry == nil {
		t.Fatal("expected entry, got nil")
	}
	if entry.ExerciseName != "Squat" {
		t.Fatalf("expected Squat, got %q", entry.ExerciseName)
	}
}

func TestEntryController_GetEntry_WrongUser(t *testing.T) {
	ec, mock := setupEntryController(t)
	mock.entries = []models.ExerciseEntry{
		{ID: 1, UserID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100},
	}

	entry, err := ec.GetEntry(1, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry != nil {
		t.Fatalf("expected nil entry for wrong user, got %+v", entry)
	}
}

func TestEntryController_CreateEntry(t *testing.T) {
	ec, mock := setupEntryController(t)

	entry, err := ec.CreateEntry(1, "Deadlift", "PR", 5, 180.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry == nil {
		t.Fatal("expected entry, got nil")
	}
	if entry.ID != 1 {
		t.Fatalf("expected ID 1, got %d", entry.ID)
	}
	if entry.UserID != 1 {
		t.Fatalf("expected UserID 1, got %d", entry.UserID)
	}
	if entry.ExerciseName != "Deadlift" {
		t.Fatalf("expected Deadlift, got %q", entry.ExerciseName)
	}
	if entry.Reps != 5 {
		t.Fatalf("expected reps 5, got %d", entry.Reps)
	}
	if entry.Weight != 180.5 {
		t.Fatalf("expected weight 180.5, got %f", entry.Weight)
	}
	if entry.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set")
	}

	// Verify it was stored
	if len(mock.entries) != 1 {
		t.Fatalf("expected 1 stored entry, got %d", len(mock.entries))
	}
}

func TestEntryController_CreateEntry_RepositoryError(t *testing.T) {
	ec, mock := setupEntryController(t)
	mock.errCreateEntry = errors.New("db error")

	_, err := ec.CreateEntry(1, "Squat", "", 5, 100)
	if err == nil {
		t.Fatal("expected error from repository")
	}
}

func TestEntryController_UpdateEntry(t *testing.T) {
	ec, mock := setupEntryController(t)
	mock.entries = []models.ExerciseEntry{
		{ID: 1, UserID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100, CreatedAt: time.Now()},
	}
	newTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	entry, err := ec.UpdateEntry(1, 1, "Squat", "Heavy", 3, 120, newTime)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry == nil {
		t.Fatal("expected entry, got nil")
	}
	if entry.Reps != 3 {
		t.Fatalf("expected reps 3, got %d", entry.Reps)
	}
	if entry.Weight != 120 {
		t.Fatalf("expected weight 120, got %f", entry.Weight)
	}
	if !entry.CreatedAt.Equal(newTime) {
		t.Fatalf("expected created_at to be updated")
	}

	// Verify stored
	stored := mock.entries[0]
	if stored.Reps != 3 {
		t.Fatalf("expected stored reps 3, got %d", stored.Reps)
	}
}

func TestEntryController_DeleteEntry(t *testing.T) {
	ec, mock := setupEntryController(t)
	mock.entries = []models.ExerciseEntry{
		{ID: 1, UserID: 1, ExerciseName: "Squat", Reps: 5, Weight: 100},
	}

	if err := ec.DeleteEntry(1, 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.entries) != 0 {
		t.Fatalf("expected entry to be deleted, got %d entries", len(mock.entries))
	}
}

func TestEntryController_ListTypes(t *testing.T) {
	ec, _ := setupEntryController(t)

	types, err := ec.ListTypes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(types) != 2 {
		t.Fatalf("expected 2 types, got %d", len(types))
	}
}

func TestEntryController_ListTypes_Error(t *testing.T) {
	ec, mock := setupEntryController(t)
	mock.errListTypes = errors.New("db error")

	_, err := ec.ListTypes()
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
