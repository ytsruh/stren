package models

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"stren/internal/db"
)

// UserRepository provides CRUD operations for users.
type UserRepository struct {
	queries *db.Queries
}

// NewUserRepository creates a new user repository.
func NewUserRepository(dbConn *db.DB) *UserRepository {
	return &UserRepository{
		queries: db.New(dbConn.Conn()),
	}
}

// CreateUser inserts a new user into the database.
func (r *UserRepository) CreateUser(user *User) error {
	ctx := context.Background()
	isAdmin := int64(0)
	if user.IsAdmin {
		isAdmin = 1
	}
	id, err := r.queries.CreateUser(ctx, db.CreateUserParams{
		Name:         user.Name,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		IsAdmin:      isAdmin,
	})
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	user.ID = id
	return nil
}

// GetUserByEmail retrieves a user by their email address.
// Returns nil if not found.
func (r *UserRepository) GetUserByEmail(email string) (*User, error) {
	ctx := context.Background()
	row, err := r.queries.GetUserByEmail(ctx, email)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return mapUser(row), nil
}

// GetUserByID retrieves a user by their ID.
// Returns nil if not found.
func (r *UserRepository) GetUserByID(id int64) (*User, error) {
	ctx := context.Background()
	row, err := r.queries.GetUserByID(ctx, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}
	return mapUser(row), nil
}

// UpdateUser updates an existing user's name.
func (r *UserRepository) UpdateUser(user *User) error {
	ctx := context.Background()
	err := r.queries.UpdateUser(ctx, db.UpdateUserParams{
		Name:      user.Name,
		UpdatedAt: sql.NullTime{Time: time.Now(), Valid: true},
		ID:        user.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

func mapUser(row db.User) *User {
	isAdmin := row.IsAdmin == 1
	return &User{
		ID:           row.ID,
		Name:         row.Name,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		IsAdmin:      isAdmin,
		CreatedAt:    nullTimeToTime(row.CreatedAt),
		UpdatedAt:    nullTimeToTime(row.UpdatedAt),
	}
}
