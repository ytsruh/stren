package controllers

import (
	"database/sql"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"stren/internal/models"
)

type mockRepository struct {
	mu        sync.Mutex
	exercises []models.Exercise
	entries   []models.ExerciseEntry

	errCreate                       error
	errGetByName                    error
	errList                         error
	errCreateEntry                  error
	errGetEntry                     error
	errUpdateEntry                  error
	errUpdateEntryWithDate          error
	errDeleteEntry                  error
	errListEntries                  error
	errGetEntriesByExercisePaginated error
	errGetEntriesByDateRange        error
	errGetExerciseByID              error
	errGetMaxWeightByExercise       error
	errGetLastSetByExercise         error
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

	for _, e := range m.exercises {
		if e.ID == entry.ExerciseID {
			entry.ExerciseName = e.Name
			break
		}
	}
	entry.ID = "entry-" + entry.ExerciseID
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

func (m *mockRepository) GetEntriesByExercisePaginated(exerciseID string, userID string, limit, offset int) ([]models.ExerciseEntry, error) {
	if m.errGetEntriesByExercisePaginated != nil {
		return nil, m.errGetEntriesByExercisePaginated
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var all []models.ExerciseEntry
	for _, e := range m.entries {
		if e.ExerciseID == exerciseID && e.UserID == userID {
			all = append(all, e)
		}
	}
	if offset >= len(all) {
		return []models.ExerciseEntry{}, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], nil
}

func (m *mockRepository) GetMaxWeightByExercise(exerciseID string, userID string) (float64, error) {
	if m.errGetMaxWeightByExercise != nil {
		return 0, m.errGetMaxWeightByExercise
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var max float64
	for _, e := range m.entries {
		if e.ExerciseID == exerciseID && e.UserID == userID && e.Weight > max {
			max = e.Weight
		}
	}
	return max, nil
}

func (m *mockRepository) GetLastSetByExercise(exerciseID string, userID string) (*models.ExerciseEntry, error) {
	if m.errGetLastSetByExercise != nil {
		return nil, m.errGetLastSetByExercise
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var last *models.ExerciseEntry
	for i, e := range m.entries {
		if e.ExerciseID != exerciseID || e.UserID != userID {
			continue
		}
		if last == nil || e.CreatedAt.After(last.CreatedAt) {
			cp := m.entries[i]
			last = &cp
		}
	}
	return last, nil
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

func (m *mockRepository) ListEntriesLast7Days(userID string) ([]models.ExerciseEntry, error) {
	if m.errListEntries != nil {
		return nil, m.errListEntries
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)
	var result []models.ExerciseEntry
	for _, e := range m.entries {
		if e.CreatedAt.After(sevenDaysAgo) && e.UserID == userID {
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

	entry, err := ec.CreateEntry("user-1", "ex-1", "great set", 5, 100, 60)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.ExerciseID != "ex-1" {
		t.Errorf("expected 'ex-1', got %q", entry.ExerciseID)
	}
	if entry.Reps != 5 {
		t.Errorf("expected 5 reps, got %d", entry.Reps)
	}
}

func TestEntryController_CreateEntry_RepositoryError(t *testing.T) {
	ec, mock := setupEntryController(t)
	mock.errCreateEntry = errors.New("db error")

	_, err := ec.CreateEntry("user-1", "ex-1", "great set", 5, 100, 60)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestEntryController_CreateEntries_Success(t *testing.T) {
	ec, mock := setupEntryController(t)

	sets := []EntrySetInput{
		{Reps: 5, Weight: 100, RestTime: 60},
		{Reps: 5, Weight: 100, RestTime: 60},
		{Reps: 5, Weight: 95, RestTime: 90},
	}

	created, err := ec.CreateEntries("user-1", "ex-1", "felt good", sets)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(created) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(created))
	}
	if len(mock.entries) != 3 {
		t.Fatalf("expected 3 entries in mock, got %d", len(mock.entries))
	}

	// All entries share the same exercise, user, notes and timestamp.
	first := created[0]
	for i, e := range created {
		if e.ExerciseID != "ex-1" {
			t.Errorf("entry %d: expected exercise 'ex-1', got %q", i, e.ExerciseID)
		}
		if e.UserID != "user-1" {
			t.Errorf("entry %d: expected user 'user-1', got %q", i, e.UserID)
		}
		if e.Notes != "felt good" {
			t.Errorf("entry %d: expected notes 'felt good', got %q", i, e.Notes)
		}
		if !e.CreatedAt.Equal(first.CreatedAt) {
			t.Errorf("entry %d: expected shared timestamp %v, got %v", i, first.CreatedAt, e.CreatedAt)
		}
	}

	// Per-set values are preserved in submission order.
	if created[2].Weight != 95 || created[2].RestTime != 90 {
		t.Errorf("third set values wrong: %+v", created[2])
	}
}

func TestEntryController_CreateEntries_EmptySets(t *testing.T) {
	ec, mock := setupEntryController(t)

	created, err := ec.CreateEntries("user-1", "ex-1", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(created) != 0 {
		t.Errorf("expected 0 entries, got %d", len(created))
	}
	if len(mock.entries) != 0 {
		t.Errorf("expected mock to be empty, got %d entries", len(mock.entries))
	}
}

func TestEntryController_CreateEntries_RepositoryErrorShortCircuits(t *testing.T) {
	ec, mock := setupEntryController(t)
	mock.errCreateEntry = errors.New("db error")

	sets := []EntrySetInput{
		{Reps: 5, Weight: 100, RestTime: 0},
		{Reps: 5, Weight: 100, RestTime: 0},
	}

	_, err := ec.CreateEntries("user-1", "ex-1", "", sets)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "db error") {
		t.Errorf("expected db error to bubble up, got %v", err)
	}
}

func TestEntryController_UpdateEntry(t *testing.T) {
	ec, _ := setupEntryController(t)
	mock := ec.repo.(*mockRepository)
	mock.entries = []models.ExerciseEntry{
		{ID: "entry-1", UserID: "user-1", ExerciseID: "ex-1", ExerciseName: "Squat", Reps: 5, Weight: 100},
	}

	entry, err := ec.UpdateEntry("entry-1", "user-1", "ex-1", "even better", 6, 110, 90, time.Now())
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
	now := time.Now()
	mock.entries = []models.ExerciseEntry{
		{ID: "entry-1", UserID: "user-1", ExerciseID: "ex-1", ExerciseName: "Squat", Reps: 5, Weight: 100, CreatedAt: now.Add(-2 * time.Hour)},
		{ID: "entry-2", UserID: "user-1", ExerciseID: "ex-2", ExerciseName: "Bench Press", Reps: 8, Weight: 80, CreatedAt: now.Add(-1 * time.Hour)},
		{ID: "entry-3", UserID: "user-1", ExerciseID: "ex-1", ExerciseName: "Squat", Reps: 3, Weight: 110, CreatedAt: now},
	}

	page, err := ec.GetEntriesByExercise("ex-1", "user-1", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Entries) != 2 {
		t.Fatalf("expected 2 squat entries, got %d", len(page.Entries))
	}
	if page.Stats.MaxWeight != 110 {
		t.Errorf("expected max weight 110, got %v", page.Stats.MaxWeight)
	}
	if page.Stats.LastSet.Weight != 110 {
		t.Errorf("expected last set weight 110, got %v", page.Stats.LastSet.Weight)
	}
}

func TestEntryController_GetEntriesByExercise_Pagination(t *testing.T) {
	ec, mock := setupEntryController(t)
	now := time.Now()
	var entries []models.ExerciseEntry
	for i := 0; i < 30; i++ {
		entries = append(entries, models.ExerciseEntry{
			ID:         "entry-" + string(rune('a'+i)),
			UserID:     "user-1",
			ExerciseID: "ex-1",
			Reps:       5,
			Weight:     float64(100 + i),
			CreatedAt:  now.Add(time.Duration(i) * time.Minute),
		})
	}
	mock.entries = entries

	page1, err := ec.GetEntriesByExercise("ex-1", "user-1", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page1.Entries) != ExerciseHistoryPageSize {
		t.Fatalf("page 1: expected %d entries, got %d", ExerciseHistoryPageSize, len(page1.Entries))
	}
	if !page1.HasNext {
		t.Error("page 1 should have a next page")
	}
	if page1.HasPrev {
		t.Error("page 1 should not have a previous page")
	}
	if page1.Page != 1 {
		t.Errorf("expected page 1, got %d", page1.Page)
	}
	// Stats should reflect the full history, not the slice.
	if page1.Stats.MaxWeight != 129 {
		t.Errorf("expected max weight 129 across all 30 entries, got %v", page1.Stats.MaxWeight)
	}

	page2, err := ec.GetEntriesByExercise("ex-1", "user-1", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page2.Entries) != 5 {
		t.Fatalf("page 2: expected 5 entries (30 - 25), got %d", len(page2.Entries))
	}
	if page2.HasNext {
		t.Error("page 2 (last page) should not have a next page")
	}
	if !page2.HasPrev {
		t.Error("page 2 should have a previous page")
	}

	page3, err := ec.GetEntriesByExercise("ex-1", "user-1", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page3.Entries) != 0 {
		t.Fatalf("page 3 (beyond data): expected 0 entries, got %d", len(page3.Entries))
	}
	if page3.HasNext {
		t.Error("page 3 (beyond data) should not have a next page")
	}
	if !page3.HasPrev {
		t.Error("page 3 (beyond data) should still have a previous page")
	}
}

func TestEntryController_GetEntriesByExercise_ClampsInvalidPage(t *testing.T) {
	ec, mock := setupEntryController(t)
	mock.entries = []models.ExerciseEntry{
		{ID: "entry-1", UserID: "user-1", ExerciseID: "ex-1", Reps: 5, Weight: 100},
	}

	page, err := ec.GetEntriesByExercise("ex-1", "user-1", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.Page != 1 {
		t.Errorf("expected page to be clamped to 1, got %d", page.Page)
	}
}

func TestEntryController_GetEntriesByExercise_EmptyStats(t *testing.T) {
	ec, _ := setupEntryController(t)

	page, err := ec.GetEntriesByExercise("ex-1", "user-1", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.Stats.MaxWeight != 0 {
		t.Errorf("expected max weight 0 for empty history, got %v", page.Stats.MaxWeight)
	}
	if page.Stats.LastSet.ID != "" {
		t.Errorf("expected zero-value last set for empty history, got %+v", page.Stats.LastSet)
	}
	if page.HasNext || page.HasPrev {
		t.Error("empty history should have no pagination state")
	}
}