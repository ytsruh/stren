package models

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
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

// UpdateUserPassword replaces the bcrypt hash for a user. Used by the
// password-reset flow after a reset token has been successfully
// consumed. Kept separate from UpdateUser so the profile-editing
// form cannot be tricked into clearing the password by omitting
// fields. Returns the same wrapped error shape as the other repo
// methods so the controller can compare with errors.Is if needed.
func (r *UserRepository) UpdateUserPassword(userID, passwordHash string) error {
	if userID == "" {
		return fmt.Errorf("failed to update password: user id is empty")
	}
	if passwordHash == "" {
		return fmt.Errorf("failed to update password: hash is empty")
	}
	ctx := context.Background()
	if err := r.queries.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		PasswordHash: passwordHash,
		UpdatedAt:    sql.NullTime{Time: time.Now(), Valid: true},
		ID:           userID,
	}); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	return nil
}

// CreateUser inserts a new user into the database.
func (r *UserRepository) CreateUser(user *User) error {
	ctx := context.Background()
	isAdmin := int64(0)
	if user.IsAdmin {
		isAdmin = 1
	}
	id, err := r.queries.CreateUser(ctx, db.CreateUserParams{
		ID:           uuid.New().String(),
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
func (r *UserRepository) GetUserByID(id string) (*User, error) {
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

// UpdateUser updates an existing user's name, target weight, and weight unit.
// The target weight is written as NULL when the user has cleared their goal,
// so the form can be reset by submitting an empty input.
func (r *UserRepository) UpdateUser(user *User) error {
	ctx := context.Background()
	err := r.queries.UpdateUser(ctx, db.UpdateUserParams{
		Name:         user.Name,
		TargetWeight: ptrToNullFloat64(user.TargetWeight),
		WeightUnit:   user.WeightUnit,
		UpdatedAt:    sql.NullTime{Time: time.Now(), Valid: true},
		ID:           user.ID,
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
		TargetWeight: nullFloat64ToPtr(row.TargetWeight),
		WeightUnit:   row.WeightUnit,
		CreatedAt:    nullTimeToTime(row.CreatedAt),
		UpdatedAt:    nullTimeToTime(row.UpdatedAt),
	}
}

// nullFloat64ToPtr converts a sql.NullFloat64 to a *float64. nil for SQL NULL,
// &value otherwise. Lets the rest of the app distinguish "no goal" from "0".
func nullFloat64ToPtr(nf sql.NullFloat64) *float64 {
	if !nf.Valid {
		return nil
	}
	v := nf.Float64
	return &v
}

// ptrToNullFloat64 converts a *float64 to a sql.NullFloat64. nil becomes
// SQL NULL, &value becomes Valid: true.
func ptrToNullFloat64(p *float64) sql.NullFloat64 {
	if p == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *p, Valid: true}
}
