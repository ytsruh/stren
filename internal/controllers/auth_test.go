package controllers

import (
	"errors"
	"sync"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"stren/internal/models"
	"stren/internal/utils"
)

type mockUserRepository struct {
	mu    sync.Mutex
	users []models.User
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{}
}

func (m *mockUserRepository) CreateUser(user *models.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.Email == user.Email {
			return errors.New("email already exists")
		}
	}
	user.ID = "user-" + user.Email
	m.users = append(m.users, *user)
	return nil
}

func (m *mockUserRepository) GetUserByEmail(email string) (*models.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.Email == email {
			cp := u
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepository) GetUserByID(id string) (*models.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.ID == id {
			cp := u
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepository) UpdateUser(user *models.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, u := range m.users {
		if u.ID == user.ID {
			m.users[i] = *user
			return nil
		}
	}
	return errors.New("user not found")
}

func setupAuthController(t *testing.T) (*AuthController, *mockUserRepository) {
	t.Helper()
	mockUser := newMockUserRepository()
	jwtService := utils.NewJWTService("test-secret")
	return NewAuthController(mockUser, jwtService), mockUser
}

func TestAuthController_Login_Success(t *testing.T) {
	ac, mockUser := setupAuthController(t)

	hash, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.DefaultCost)
	mockUser.users = append(mockUser.users, models.User{
		ID:           "user-1",
		Name:         "Alice",
		Email:        "alice@example.com",
		PasswordHash: string(hash),
	})

	user, token, err := ac.Login("alice@example.com", "correct")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user == nil {
		t.Fatal("expected user, got nil")
	}
	if user.Email != "alice@example.com" {
		t.Fatalf("expected email alice@example.com, got %q", user.Email)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestAuthController_Login_InvalidPassword(t *testing.T) {
	ac, mockUser := setupAuthController(t)

	hash, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.DefaultCost)
	mockUser.users = append(mockUser.users, models.User{
		ID:           "user-1",
		Name:         "Bob",
		Email:        "bob@example.com",
		PasswordHash: string(hash),
	})

	_, _, err := ac.Login("bob@example.com", "wrong")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthController_Login_UserNotFound(t *testing.T) {
	ac, _ := setupAuthController(t)

	_, _, err := ac.Login("nobody@example.com", "password")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthController_Login_EmptyFields(t *testing.T) {
	ac, _ := setupAuthController(t)

	tests := []struct {
		email    string
		password string
	}{
		{"", "password"},
		{"email@example.com", ""},
		{"", ""},
	}

	for _, tt := range tests {
		_, _, err := ac.Login(tt.email, tt.password)
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials for email=%q password=%q, got %v", tt.email, tt.password, err)
		}
	}
}

func TestAuthController_Register_Success(t *testing.T) {
	ac, mockUser := setupAuthController(t)

	user, token, err := ac.Register("Alice", "alice@example.com", "secret123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user == nil {
		t.Fatal("expected user, got nil")
	}
	if user.ID == "" {
		t.Fatalf("expected non-empty user ID")
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	stored := mockUser.users[0]
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte("secret123")); err != nil {
		t.Fatal("expected stored password to be a valid bcrypt hash")
	}
}

func TestAuthController_Register_EmailExists(t *testing.T) {
	ac, mockUser := setupAuthController(t)

	mockUser.users = append(mockUser.users, models.User{
		ID:    "user-1",
		Name:  "Alice",
		Email: "alice@example.com",
	})

	_, _, err := ac.Register("Alice2", "alice@example.com", "secret123")
	if !errors.Is(err, ErrEmailExists) {
		t.Fatalf("expected ErrEmailExists, got %v", err)
	}
}

func TestAuthController_Register_EmptyFields(t *testing.T) {
	ac, _ := setupAuthController(t)

	tests := []struct {
		name     string
		email    string
		password string
	}{
		{"", "email@example.com", "password"},
		{"Name", "", "password"},
		{"Name", "email@example.com", ""},
		{"", "", ""},
	}

	for _, tt := range tests {
		_, _, err := ac.Register(tt.name, tt.email, tt.password)
		if err == nil {
			t.Fatalf("expected error for name=%q email=%q", tt.name, tt.email)
		}
		if err.Error() != "all fields are required" {
			t.Fatalf("expected 'all fields are required', got %q", err.Error())
		}
	}
}

func TestAuthController_Register_ShortPassword(t *testing.T) {
	ac, _ := setupAuthController(t)

	_, _, err := ac.Register("Alice", "alice@example.com", "short")
	if err == nil {
		t.Fatal("expected error for short password")
	}
	if err.Error() != "password must be at least 6 characters" {
		t.Fatalf("expected password length error, got %q", err.Error())
	}
}

func TestAuthController_RegisterAndLogin(t *testing.T) {
	ac, _ := setupAuthController(t)

	user, token, err := ac.Register("Alice", "alice@example.com", "secret123")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if user == nil || token == "" {
		t.Fatal("expected user and token after registration")
	}

	loggedInUser, loginToken, err := ac.Login("alice@example.com", "secret123")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if loggedInUser == nil || loginToken == "" {
		t.Fatal("expected user and token after login")
	}
	if loggedInUser.Email != "alice@example.com" {
		t.Fatalf("expected email alice@example.com, got %q", loggedInUser.Email)
	}
}