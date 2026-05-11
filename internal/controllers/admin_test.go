package controllers

import (
	"errors"
	"sync"
	"testing"

	"stren/internal/models"
)

// mockAdminRepository is an in-memory implementation of models.AdminRepository for testing.
type mockAdminRepository struct {
	mu        sync.Mutex
	exercises []models.Exercise
	nextID    int64

	errGetByID    error
	errUpdate     error
	errCreateNoTx error
	errList       error
}

func newMockAdminRepository() *mockAdminRepository {
	return &mockAdminRepository{
		nextID: 1,
	}
}

func (m *mockAdminRepository) GetByID(id int64) (*models.Exercise, error) {
	if m.errGetByID != nil {
		return nil, m.errGetByID
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

func (m *mockAdminRepository) Update(id int64, name string) (*models.Exercise, error) {
	if m.errUpdate != nil {
		return nil, m.errUpdate
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, e := range m.exercises {
		if e.ID == id {
			m.exercises[i].Name = name
			cp := m.exercises[i]
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockAdminRepository) CreateNoTx(name string) (int64, error) {
	if m.errCreateNoTx != nil {
		return 0, m.errCreateNoTx
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.exercises {
		if e.Name == name {
			return e.ID, nil
		}
	}
	id := m.nextID
	m.nextID++
	m.exercises = append(m.exercises, models.Exercise{ID: id, Name: name})
	return id, nil
}

func (m *mockAdminRepository) List() ([]models.Exercise, error) {
	if m.errList != nil {
		return nil, m.errList
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]models.Exercise, len(m.exercises))
	copy(result, m.exercises)
	return result, nil
}

func TestAdminController_List(t *testing.T) {
	mock := newMockAdminRepository()
	mock.exercises = []models.Exercise{
		{ID: 1, Name: "Squat"},
		{ID: 2, Name: "Bench Press"},
	}

	ctrl := NewAdminController(mock)
	exercises, err := ctrl.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(exercises) != 2 {
		t.Fatalf("expected 2 exercises, got %d", len(exercises))
	}
}

func TestAdminController_List_Error(t *testing.T) {
	mock := newMockAdminRepository()
	mock.errList = errors.New("database error")

	ctrl := NewAdminController(mock)
	_, err := ctrl.List()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAdminController_Get(t *testing.T) {
	mock := newMockAdminRepository()
	mock.exercises = []models.Exercise{
		{ID: 1, Name: "Squat"},
		{ID: 2, Name: "Bench Press"},
	}

	ctrl := NewAdminController(mock)

	t.Run("found", func(t *testing.T) {
		ex, err := ctrl.Get(1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ex.Name != "Squat" {
			t.Errorf("expected 'Squat', got %q", ex.Name)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := ctrl.Get(999)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		mock.errGetByID = errors.New("database error")
		_, err := ctrl.Get(1)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		mock.errGetByID = nil
	})
}

func TestAdminController_Create(t *testing.T) {
	mock := newMockAdminRepository()
	ctrl := NewAdminController(mock)

	t.Run("success", func(t *testing.T) {
		ex, err := ctrl.Create("Deadlift")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ex.Name != "Deadlift" {
			t.Errorf("expected 'Deadlift', got %q", ex.Name)
		}
	})

	t.Run("empty name", func(t *testing.T) {
		_, err := ctrl.Create("   ")
		if err == nil {
			t.Fatal("expected error for empty name, got nil")
		}
	})

	t.Run("repository error", func(t *testing.T) {
		mock.errCreateNoTx = errors.New("database error")
		_, err := ctrl.Create("Squat")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		mock.errCreateNoTx = nil
	})
}

func TestAdminController_Update(t *testing.T) {
	mock := newMockAdminRepository()
	mock.exercises = []models.Exercise{
		{ID: 1, Name: "Squat"},
	}

	ctrl := NewAdminController(mock)

	t.Run("success", func(t *testing.T) {
		ex, err := ctrl.Update(1, "Front Squat")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ex.Name != "Front Squat" {
			t.Errorf("expected 'Front Squat', got %q", ex.Name)
		}
	})

	t.Run("empty name", func(t *testing.T) {
		_, err := ctrl.Update(1, "   ")
		if err == nil {
			t.Fatal("expected error for empty name, got nil")
		}
	})

	t.Run("repository error", func(t *testing.T) {
		mock.errUpdate = errors.New("database error")
		_, err := ctrl.Update(1, "Squat")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		mock.errUpdate = nil
	})
}