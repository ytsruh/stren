package models

import (
	"context"
	"fmt"

	"stren/internal/db"
)

// UserAdminRepository provides admin-only user operations.
type UserAdminRepository struct {
	queries *db.Queries
}

// NewUserAdminRepository creates a new admin user repository.
func NewUserAdminRepository(dbConn *db.DB) *UserAdminRepository {
	return &UserAdminRepository{
		queries: db.New(dbConn.Conn()),
	}
}

// ListUsers retrieves all users ordered by creation date (newest first).
func (r *UserAdminRepository) ListUsers() ([]User, error) {
	ctx := context.Background()
	rows, err := r.queries.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	users := make([]User, len(rows))
	for i, row := range rows {
		users[i] = *mapUser(row)
	}
	return users, nil
}
