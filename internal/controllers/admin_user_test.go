package controllers

import (
	"sync"
	"testing"

	"stren/internal/models"
)

// mockAdminUserRepository is an in-memory implementation of models.AdminUserRepo for testing.
type mockAdminUserRepository struct {
	mu    sync.Mutex
	users []models.User
}

func newMockAdminUserRepository() *mockAdminUserRepository {
	return &mockAdminUserRepository{}
}

func (m *mockAdminUserRepository) ListUsers() ([]models.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]models.User, len(m.users))
	copy(result, m.users)
	return result, nil
}

func TestAdminUserController_ListUsers(t *testing.T) {
	mock := newMockAdminUserRepository()
	mock.users = []models.User{
		{ID: 1, Name: "Alice", Email: "alice@example.com", IsAdmin: true},
		{ID: 2, Name: "Bob", Email: "bob@example.com", IsAdmin: false},
	}

	ctrl := NewAdminUserController(mock)
	users, err := ctrl.ListUsers()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}

	if users[0].Name != "Alice" {
		t.Errorf("expected first user to be 'Alice', got %q", users[0].Name)
	}
}

func TestAdminUserController_ListUsers_Empty(t *testing.T) {
	mock := newMockAdminUserRepository()
	ctrl := NewAdminUserController(mock)

	users, err := ctrl.ListUsers()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(users) != 0 {
		t.Fatalf("expected 0 users, got %d", len(users))
	}
}

func TestAdminUserController_ListUsers_Error(t *testing.T) {
	mock := newMockAdminUserRepository()
	ctrl := NewAdminUserController(mock)

	_, err := ctrl.ListUsers()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}