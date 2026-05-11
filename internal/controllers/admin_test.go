package controllers

import (
	"errors"
	"sync"
	"testing"

	"stren/internal/models"
)

// mockAdminRepository is an in-memory implementation of models.AdminRepository for testing.
type mockAdminRepository struct {
	mu      sync.Mutex
	types   []models.ExerciseType
	nextID  int64

	errGetTypeByID    error
	errUpdateType     error
	errCreateTypeNoTx error
	errListTypes      error
}

func newMockAdminRepository() *mockAdminRepository {
	return &mockAdminRepository{
		nextID: 1,
	}
}

func (m *mockAdminRepository) GetTypeByID(id int64) (*models.ExerciseType, error) {
	if m.errGetTypeByID != nil {
		return nil, m.errGetTypeByID
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.types {
		if t.ID == id {
			cp := t
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockAdminRepository) UpdateType(id int64, name string) (*models.ExerciseType, error) {
	if m.errUpdateType != nil {
		return nil, m.errUpdateType
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, t := range m.types {
		if t.ID == id {
			m.types[i].Name = name
			cp := m.types[i]
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockAdminRepository) CreateTypeNoTx(name string) (int64, error) {
	if m.errCreateTypeNoTx != nil {
		return 0, m.errCreateTypeNoTx
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.types {
		if t.Name == name {
			return t.ID, nil
		}
	}
	id := m.nextID
	m.nextID++
	m.types = append(m.types, models.ExerciseType{ID: id, Name: name})
	return id, nil
}

func (m *mockAdminRepository) ListTypes() ([]models.ExerciseType, error) {
	if m.errListTypes != nil {
		return nil, m.errListTypes
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]models.ExerciseType, len(m.types))
	copy(result, m.types)
	return result, nil
}

func TestAdminController_ListTypes(t *testing.T) {
	mock := newMockAdminRepository()
	mock.types = []models.ExerciseType{
		{ID: 1, Name: "Squat"},
		{ID: 2, Name: "Bench Press"},
	}

	ctrl := NewAdminController(mock)
	types, err := ctrl.ListTypes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(types) != 2 {
		t.Fatalf("expected 2 types, got %d", len(types))
	}
}

func TestAdminController_ListTypes_Error(t *testing.T) {
	mock := newMockAdminRepository()
	mock.errListTypes = errors.New("database error")

	ctrl := NewAdminController(mock)
	_, err := ctrl.ListTypes()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAdminController_GetType(t *testing.T) {
	mock := newMockAdminRepository()
	mock.types = []models.ExerciseType{
		{ID: 1, Name: "Squat"},
		{ID: 2, Name: "Bench Press"},
	}

	ctrl := NewAdminController(mock)

	t.Run("found", func(t *testing.T) {
		ex, err := ctrl.GetType(1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ex.Name != "Squat" {
			t.Errorf("expected 'Squat', got %q", ex.Name)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := ctrl.GetType(999)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		mock.errGetTypeByID = errors.New("database error")
		_, err := ctrl.GetType(1)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		mock.errGetTypeByID = nil
	})
}

func TestAdminController_CreateType(t *testing.T) {
	mock := newMockAdminRepository()
	ctrl := NewAdminController(mock)

	t.Run("success", func(t *testing.T) {
		ex, err := ctrl.CreateType("Deadlift")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ex.Name != "Deadlift" {
			t.Errorf("expected 'Deadlift', got %q", ex.Name)
		}
	})

	t.Run("empty name", func(t *testing.T) {
		_, err := ctrl.CreateType("   ")
		if err == nil {
			t.Fatal("expected error for empty name, got nil")
		}
	})

	t.Run("repository error", func(t *testing.T) {
		mock.errCreateTypeNoTx = errors.New("database error")
		_, err := ctrl.CreateType("Squat")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		mock.errCreateTypeNoTx = nil
	})
}

func TestAdminController_UpdateType(t *testing.T) {
	mock := newMockAdminRepository()
	mock.types = []models.ExerciseType{
		{ID: 1, Name: "Squat"},
	}

	ctrl := NewAdminController(mock)

	t.Run("success", func(t *testing.T) {
		ex, err := ctrl.UpdateType(1, "Front Squat")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ex.Name != "Front Squat" {
			t.Errorf("expected 'Front Squat', got %q", ex.Name)
		}
	})

	t.Run("empty name", func(t *testing.T) {
		_, err := ctrl.UpdateType(1, "   ")
		if err == nil {
			t.Fatal("expected error for empty name, got nil")
		}
	})

	t.Run("repository error", func(t *testing.T) {
		mock.errUpdateType = errors.New("database error")
		_, err := ctrl.UpdateType(1, "Squat")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		mock.errUpdateType = nil
	})
}