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
// The context is passed through to the underlying sqlc query so a
// caller-driven cancellation (e.g. the weekly reminder's parent
// context) propagates to the database. Production code that does
// not have a request context to thread through can pass
// context.Background().
func (r *UserAdminRepository) ListUsers(ctx context.Context) ([]User, error) {
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
